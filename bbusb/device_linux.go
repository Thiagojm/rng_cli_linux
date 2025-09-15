//go:build linux

package bbusb

import (
	"context"
	"fmt"
	"time"

	"github.com/google/gousb"
)

// DeviceSession encapsulates a libusb session to BitBabbler on Linux.
type DeviceSession struct {
	ctx  *gousb.Context
	dev  *gousb.Device
	intf *gousb.Interface
	inEp *gousb.InEndpoint
}

// OpenBitBabbler opens BitBabbler using libusb and prepares a bulk IN endpoint for reads.
func OpenBitBabbler(_ uint, _ uint8) (*DeviceSession, error) {
	ctx := gousb.NewContext()

	dev, err := ctx.OpenDeviceWithVIDPID(gousb.ID(ftdiVendorID), gousb.ID(bbProductID))
	if err != nil {
		ctx.Close()
		return nil, fmt.Errorf("open usb device: %w", err)
	}
	if dev == nil {
		ctx.Close()
		return nil, fmt.Errorf("BitBabbler device not found")
	}

	intf, done, err := dev.DefaultInterface()
	if err != nil {
		dev.Close()
		ctx.Close()
		return nil, fmt.Errorf("claim interface: %w", err)
	}

	var inEp *gousb.InEndpoint
	for _, ep := range intf.Setting.Endpoints {
		if ep.TransferType == gousb.TransferTypeBulk && ep.Direction == gousb.EndpointDirectionIn {
			inEp, err = intf.InEndpoint(ep.Number)
			if err != nil {
				done()
				dev.Close()
				ctx.Close()
				return nil, fmt.Errorf("open in endpoint: %w", err)
			}
			break
		}
	}
	if inEp == nil {
		done()
		dev.Close()
		ctx.Close()
		return nil, fmt.Errorf("bulk IN endpoint not found")
	}

	// Keep interface claimed; release on Close.
	return &DeviceSession{ctx: ctx, dev: dev, intf: intf, inEp: inEp}, nil
}

// Close releases USB resources.
func (s *DeviceSession) Close() {
	if s == nil {
		return
	}
	if s.intf != nil {
		s.intf.Close()
	}
	if s.dev != nil {
		s.dev.Close()
	}
	if s.ctx != nil {
		s.ctx.Close()
	}
}

// ReadRandom reads random data via the bulk IN endpoint.
func (s *DeviceSession) ReadRandom(buf []byte) (int, error) {
	if len(buf) == 0 {
		return 0, nil
	}
	// Give the device a short settle
	_ = s.dev.SetAutoDetach(true)
	time.Sleep(20 * time.Millisecond)

	total := 0
	deadline := time.Now().Add(5 * time.Second)
	tmp := make([]byte, 4096)
	for total < len(buf) {
		if time.Now().After(deadline) {
			break
		}
		n, err := s.inEp.Read(tmp)
		if err != nil {
			return total, err
		}
		if n > 0 {
			toCopy := n
			if toCopy > len(buf)-total {
				toCopy = len(buf) - total
			}
			copy(buf[total:total+toCopy], tmp[:toCopy])
			total += toCopy
		} else {
			time.Sleep(5 * time.Millisecond)
		}
	}
	return total, nil
}

// Unused to keep API parity on Linux build
func (s *DeviceSession) ReadRandomWithContext(_ context.Context, buf []byte) (int, error) {
	return s.ReadRandom(buf)
}
