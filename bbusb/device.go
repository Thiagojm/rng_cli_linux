//go:build !linux

package bbusb

import (
	"fmt"
	"time"

	"go.bug.st/serial"
)

// DeviceSession encapsulates an open BitBabbler device via serial connection.
type DeviceSession struct {
	port     serial.Port
	portName string
}

// OpenBitBabbler opens the first BitBabbler device as a serial device.
// This uses the FTDI serial driver that should be loaded by our udev rules.
func OpenBitBabbler(bitrate uint, latencyMs uint8) (*DeviceSession, error) {
	// Find the BitBabbler device
	device, err := FindDevice()
	if err != nil {
		return nil, fmt.Errorf("BitBabbler device not found: %w", err)
	}

	// Set up serial mode - use standard baud rate for FTDI serial mode
	mode := &serial.Mode{
		BaudRate: 115200, // Standard baud rate for FTDI serial mode
		Parity:   serial.NoParity,
		DataBits: 8,
		StopBits: serial.OneStopBit,
	}

	// Try to open the device
	port, err := serial.Open(device.DevicePath, mode)
	if err != nil {
		return nil, fmt.Errorf("failed to open serial port %s: %w", device.DevicePath, err)
	}

	session := &DeviceSession{
		port:     port,
		portName: device.DevicePath,
	}

	// Basic initialization - set DTR and flush
	if err := port.SetDTR(true); err != nil {
		session.Close()
		return nil, fmt.Errorf("failed to set DTR: %w", err)
	}

	time.Sleep(100 * time.Millisecond)

	if err := port.ResetInputBuffer(); err != nil {
		session.Close()
		return nil, fmt.Errorf("failed to reset input buffer: %w", err)
	}

	return session, nil
}

// Close releases the serial port.
func (s *DeviceSession) Close() {
	if s != nil && s.port != nil {
		s.port.Close()
	}
}

// ReadRandom reads random data from the BitBabbler device.
// This is a simplified implementation that works with the serial interface.
func (s *DeviceSession) ReadRandom(buf []byte) (int, error) {
	if len(buf) == 0 {
		return 0, nil
	}

	// For BitBabbler devices, we can read data directly from the serial port
	// The device should provide random data continuously

	total := 0
	deadline := time.Now().Add(5 * time.Second)

	for total < len(buf) {
		if time.Now().After(deadline) {
			break // Timeout
		}

		n, err := s.port.Read(buf[total:])
		if err != nil {
			return total, fmt.Errorf("serial read error: %w", err)
		}

		total += n
		if n == 0 {
			time.Sleep(10 * time.Millisecond)
		}
	}

	return total, nil
}
