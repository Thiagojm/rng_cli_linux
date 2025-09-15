//go:build linux

package bbusb

import (
	"fmt"
	"time"

	"github.com/google/gousb"
)

// DeviceSession encapsulates an open BitBabbler FTDI device via gousb (Linux).
type DeviceSession struct {
	ctx       *gousb.Context
	dev       *gousb.Device
	cfg       *gousb.Config
	intf      *gousb.Interface
	inEp      *gousb.InEndpoint
	outEp     *gousb.OutEndpoint
	maxPacket int
}

// OpenBitBabbler opens the BitBabbler device and initializes MPSSE like the Windows implementation.
func OpenBitBabbler(bitrate uint, latencyMs uint8) (*DeviceSession, error) {
	if bitrate == 0 {
		bitrate = 2_500_000
	}
	if latencyMs == 0 {
		latencyMs = 1
	}

	ctx := gousb.NewContext()

	dev, err := ctx.OpenDeviceWithVIDPID(gousb.ID(ftdiVendorID), gousb.ID(bbProductID))
	if err != nil {
		ctx.Close()
		return nil, err
	}
	if dev == nil {
		ctx.Close()
		return nil, fmt.Errorf("BitBabbler device not found")
	}

	_ = dev.SetAutoDetach(true)

	cfg, err := dev.Config(1)
	if err != nil {
		dev.Close()
		ctx.Close()
		return nil, err
	}
	intf, err := cfg.Interface(0, 0)
	if err != nil {
		cfg.Close()
		dev.Close()
		ctx.Close()
		return nil, err
	}

	var inEp *gousb.InEndpoint
	var outEp *gousb.OutEndpoint
	for _, ep := range intf.Setting.Endpoints {
		if ep.Direction == gousb.EndpointDirectionIn && ep.TransferType == gousb.TransferTypeBulk {
			inEp, err = intf.InEndpoint(ep.Number)
			if err != nil {
				intf.Close()
				cfg.Close()
				dev.Close()
				ctx.Close()
				return nil, err
			}
		}
		if ep.Direction == gousb.EndpointDirectionOut && ep.TransferType == gousb.TransferTypeBulk {
			outEp, err = intf.OutEndpoint(ep.Number)
			if err != nil {
				intf.Close()
				cfg.Close()
				dev.Close()
				ctx.Close()
				return nil, err
			}
		}
	}
	if inEp == nil || outEp == nil {
		intf.Close()
		cfg.Close()
		dev.Close()
		ctx.Close()
		return nil, fmt.Errorf("bulk endpoints not found")
	}

	s := &DeviceSession{ctx: ctx, dev: dev, cfg: cfg, intf: intf, inEp: inEp, outEp: outEp, maxPacket: int(inEp.Desc.MaxPacketSize)}

	// FTDI/MPSSE init
	if err := s.ftdiReset(); err != nil {
		s.Close()
		return nil, err
	}
	if err := s.purgeRead(); err != nil {
		s.Close()
		return nil, err
	}
	if err := s.ftdiSetSpecialChars(0, false, 0, false); err != nil {
		s.Close()
		return nil, err
	}
	if err := s.ftdiSetLatencyTimer(latencyMs); err != nil {
		s.Close()
		return nil, err
	}
	if err := s.ftdiSetFlowControl(ftdiFlowRtsCts); err != nil {
		s.Close()
		return nil, err
	}
	if err := s.ftdiSetBitmode(ftdiBitmodeReset, 0); err != nil {
		s.Close()
		return nil, err
	}
	if err := s.ftdiSetBitmode(ftdiBitmodeMpsse, 0); err != nil {
		s.Close()
		return nil, err
	}
	time.Sleep(50 * time.Millisecond)

	ok := s.checkSync(0xAA) && s.checkSync(0xAB)
	if !ok {
		ok = s.checkSync(0xAA) && s.checkSync(0xAB)
	}
	if !ok {
		s.Close()
		return nil, fmt.Errorf("MPSSE sync failed")
	}

	clkDiv := uint16(30000000/bitrate - 1)
	cmd := []byte{
		mpsseNoClkDiv5,
		mpsseNoAdaptiveClk,
		mpsseNo3PhaseClk,
		mpsseSetDataLow,
		0x00,
		0x0B,
		mpsseSetDataHigh,
		0x00,
		0x00,
		mpsseSetClkDivisor,
		byte(clkDiv & 0xFF),
		byte(clkDiv >> 8),
		0x85,
	}
	if _, err := s.outEp.Write(cmd); err != nil {
		s.Close()
		return nil, err
	}
	time.Sleep(30 * time.Millisecond)
	_ = s.purgeRead()

	return s, nil
}

// Close releases USB resources.
func (s *DeviceSession) Close() {
	if s == nil {
		return
	}
	if s.intf != nil {
		s.intf.Close()
	}
	if s.cfg != nil {
		s.cfg.Close()
	}
	if s.dev != nil {
		s.dev.Close()
	}
	if s.ctx != nil {
		s.ctx.Close()
	}
}

// ReadRandom issues an MPSSE read and strips FTDI status headers.
func (s *DeviceSession) ReadRandom(buf []byte) (int, error) {
	if len(buf) == 0 {
		return 0, nil
	}
	n := len(buf)
	cmd := []byte{
		mpsseDataByteInPosMSB,
		byte((n - 1) & 0xFF),
		byte((n - 1) >> 8),
		mpsseSendImmediate,
	}
	if _, err := s.outEp.Write(cmd); err != nil {
		return 0, err
	}

	want := n
	got := 0
	tmp := make([]byte, roundUpToMaxPacket(n, s.maxPacket)+s.maxPacket)
	for got < want {
		m, err := s.inEp.Read(tmp)
		if err != nil {
			return got, err
		}
		if m <= 2 {
			continue
		}
		offset := 0
		for offset < m {
			remain := m - offset
			if remain <= 2 {
				break
			}
			take := remain
			if take > s.maxPacket {
				take = s.maxPacket
			}
			usable := take - 2
			if usable > (want - got) {
				usable = want - got
			}
			copy(buf[got:got+usable], tmp[offset+2:offset+2+usable])
			got += usable
			offset += take
			if got == want {
				break
			}
		}
	}
	return got, nil
}

// ---- Helpers (FTDI control & init) ----

func (s *DeviceSession) control(req uint8, value uint16, index uint16, data []byte, in bool) error {
	typ := uint8(gousb.ControlOut) | uint8(gousb.ControlVendor) | uint8(gousb.ControlDevice)
	if in {
		typ = uint8(gousb.ControlIn) | uint8(gousb.ControlVendor) | uint8(gousb.ControlDevice)
	}
	_, err := s.dev.Control(uint8(typ), req, value, index, data)
	return err
}
func (s *DeviceSession) ftdiReset() error {
	return s.control(ftdiReqReset, ftdiResetSIO, 1, nil, false)
}
func (s *DeviceSession) ftdiSetBitmode(mode uint16, mask uint8) error {
	return s.control(ftdiReqSetBitmode, mode|uint16(mask), 1, nil, false)
}
func (s *DeviceSession) ftdiSetLatencyTimer(ms uint8) error {
	return s.control(ftdiReqSetLatency, uint16(ms), 1, nil, false)
}
func (s *DeviceSession) ftdiSetFlowControl(mode uint16) error {
	return s.control(ftdiReqSetFlowCtrl, 0, mode|1, nil, false)
}
func (s *DeviceSession) ftdiSetSpecialChars(event byte, evtEnable bool, errc byte, errEnable bool) error {
	v := uint16(event)
	if evtEnable {
		v |= 0x0100
	}
	if err := s.control(ftdiReqSetEventChar, v, 1, nil, false); err != nil {
		return err
	}
	v = uint16(errc)
	if errEnable {
		v |= 0x0100
	}
	return s.control(ftdiReqSetErrorChar, v, 1, nil, false)
}
func (s *DeviceSession) purgeRead() error {
	buf := make([]byte, 8192)
	for i := 0; i < 10; i++ {
		n, _ := s.inEp.Read(buf)
		if n <= 2 {
			break
		}
	}
	return nil
}
func (s *DeviceSession) checkSync(cmd byte) bool {
	msg := []byte{cmd, mpsseSendImmediate}
	if _, err := s.outEp.Write(msg); err != nil {
		return false
	}
	buf := make([]byte, 512)
	for i := 0; i < 10; i++ {
		n, _ := s.inEp.Read(buf)
		if n == 4 && buf[2] == 0xFA && buf[3] == cmd {
			return true
		}
	}
	return false
}
func roundUpToMaxPacket(n, max int) int {
	if max <= 0 {
		return n
	}
	if n%max == 0 {
		return n
	}
	return (n/max + 1) * max
}
