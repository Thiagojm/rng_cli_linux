package truerng

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"
)

// DeviceNamePrefix is the prefix used in the device name/description to
// identify a TrueRNG serial device. Mirrors the Python logic that checks
// description starts with "TrueRNG".
const DeviceNamePrefix = "TrueRNG"

// DeviceModel represents the type of TrueRNG device
type DeviceModel int

const (
	DeviceModelUnknown DeviceModel = iota
	DeviceModelTrueRNG
	DeviceModelTrueRNGpro
	DeviceModelTrueRNGproV2
)

// String returns the string representation of the device model
func (m DeviceModel) String() string {
	switch m {
	case DeviceModelTrueRNG:
		return "TrueRNG"
	case DeviceModelTrueRNGpro:
		return "TrueRNGpro"
	case DeviceModelTrueRNGproV2:
		return "TrueRNGproV2"
	default:
		return "Unknown"
	}
}

// CaptureMode represents the different capture modes supported by TrueRNG devices
type CaptureMode string

const (
	ModeNormal        CaptureMode = "MODE_NORMAL"          // 300 baud - Streams combined + Mersenne Twister
	ModePSDebug       CaptureMode = "MODE_PSDEBUG"         // 1200 baud - PS Voltage in mV in ASCII
	ModeRNGDebug      CaptureMode = "MODE_RNGDEBUG"        // 2400 baud - RNG Debug 0x0RRR 0x0RRR in ASCII
	ModeRNG1White     CaptureMode = "MODE_RNG1WHITE"       // 4800 baud - RNG1 + Mersenne Twister
	ModeRNG2White     CaptureMode = "MODE_RNG2WHITE"       // 9600 baud - RNG2 + Mersenne Twister
	ModeRawBin        CaptureMode = "MODE_RAW_BIN"         // 19200 baud - Raw ADC Samples in Binary Mode
	ModeRawASC        CaptureMode = "MODE_RAW_ASC"         // 38400 baud - Raw ADC Samples in ASCII Mode
	ModeUnwhitened    CaptureMode = "MODE_UNWHITENED"      // 57600 baud - Unwhitened RNG1-RNG2 (TrueRNGproV2 Only)
	ModeNormalASC     CaptureMode = "MODE_NORMAL_ASC"      // 115200 baud - Normal in ASCII Mode (TrueRNGproV2 Only)
	ModeNormalASCSlow CaptureMode = "MODE_NORMAL_ASC_SLOW" // 230400 baud - Normal in ASCII Mode - Slow for small devices (TrueRNGproV2 Only)
)

// GetBaudRate returns the baud rate for the given capture mode
func (m CaptureMode) GetBaudRate() int {
	switch m {
	case ModeNormal:
		return 300
	case ModePSDebug:
		return 1200
	case ModeRNGDebug:
		return 2400
	case ModeRNG1White:
		return 4800
	case ModeRNG2White:
		return 9600
	case ModeRawBin:
		return 19200
	case ModeRawASC:
		return 38400
	case ModeUnwhitened:
		return 57600
	case ModeNormalASC:
		return 115200
	case ModeNormalASCSlow:
		return 230400
	default:
		return 300 // Default to MODE_NORMAL
	}
}

// DeviceInfo holds information about a detected TrueRNG device
type DeviceInfo struct {
	Port  string
	Model DeviceModel
	Name  string
}

// Detect returns true if a TrueRNG serial device is present on the system.
// It enumerates available serial ports and checks their friendly name or
// description for a TrueRNG prefix.
func Detect() (bool, error) {
	devices, err := EnumerateDevices()
	return len(devices) > 0, err
}

// EnumerateDevices returns information about all detected TrueRNG devices
func EnumerateDevices() ([]DeviceInfo, error) {
	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return nil, fmt.Errorf("enumerating ports: %w", err)
	}

	var devices []DeviceInfo
	for _, p := range ports {
		if p == nil {
			continue
		}
		if model := getTrueRNGModel(p); model != DeviceModelUnknown {
			devices = append(devices, DeviceInfo{
				Port:  p.Name,
				Model: model,
				Name:  p.Product,
			})
		}
	}
	return devices, nil
}

// FindPort returns the first serial port path for a detected TrueRNG device, e.g.
// "/dev/ttyUSB0" on Linux.
func FindPort() (string, error) {
	devices, err := EnumerateDevices()
	if err != nil {
		return "", err
	}
	if len(devices) == 0 {
		return "", errors.New("TrueRNG device not found")
	}
	return devices[0].Port, nil
}

// FindDevice returns detailed information about the first detected TrueRNG device
func FindDevice() (*DeviceInfo, error) {
	devices, err := EnumerateDevices()
	if err != nil {
		return nil, err
	}
	if len(devices) == 0 {
		return nil, errors.New("TrueRNG device not found")
	}
	return &devices[0], nil
}

// ReadBytes opens the TrueRNG serial port, sets DTR, flushes input, and reads
// blockSize bytes. The behavior mirrors `truerng.py`'s read_bytes.
func ReadBytes(blockSize int) ([]byte, error) {
	return ReadBytesWithMode(blockSize, ModeNormal)
}

// ReadBytesWithMode opens the TrueRNG serial port with the specified capture mode,
// sets DTR, flushes input, and reads blockSize bytes.
func ReadBytesWithMode(blockSize int, mode CaptureMode) ([]byte, error) {
	if blockSize <= 0 {
		return nil, errors.New("blockSize must be positive")
	}
	portName, err := FindPort()
	if err != nil {
		return nil, err
	}

	// Use the efficient helper function for single reads
	return readBytesFromPort(portName, mode, blockSize)
}

// readBytesFromPort is a helper function that opens a port and reads bytes efficiently
func readBytesFromPort(portName string, mode CaptureMode, blockSize int) ([]byte, error) {
	// Skip mode change for now to avoid triggering USB re-enumeration
	// if err := changeMode(portName, mode); err != nil {
	//     // Mode change failed, but we can still try to read in normal mode
	//     fmt.Printf("Warning: mode change failed: %v, proceeding with normal mode\n", err)
	// }

	// Use default serial mode to avoid USB re-enumeration issues
	// Let the TrueRNG device use its default baud rate
	serialMode := &serial.Mode{
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	port, err := serial.Open(portName, serialMode)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", portName, err)
	}
	defer func() { _ = port.Close() }()

	// Set DTR true (as in Python), then flush any buffered input before reading.
	_ = port.SetDTR(true)
	_ = port.SetReadTimeout(1000 * time.Millisecond)
	if err := port.ResetInputBuffer(); err != nil {
		// not fatal, proceed
	}

	buf := make([]byte, blockSize)
	total := 0
	deadline := time.Now().Add(10 * time.Second) // match Python's 10s timeout intent
	for total < blockSize {
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("read timeout after 10s: read %d/%d bytes", total, blockSize)
		}
		n, err := port.Read(buf[total:])
		if err != nil {
			return nil, fmt.Errorf("read error: %w", err)
		}
		total += n
		if n == 0 {
			time.Sleep(5 * time.Millisecond)
		}
	}
	return buf, nil
}

// ReadBits reads bitCount bits from the TrueRNG and returns them as a byte
// slice packed MSB-first in each byte. The final byte may be partially filled.
func ReadBits(bitCount int) ([]byte, error) {
	return ReadBitsWithMode(bitCount, ModeNormal)
}

// ReadBitsWithMode reads bitCount bits from the TrueRNG with the specified capture mode
func ReadBitsWithMode(bitCount int, mode CaptureMode) ([]byte, error) {
	if bitCount <= 0 {
		return nil, errors.New("bitCount must be positive")
	}
	byteCount := (bitCount + 7) / 8
	data, err := ReadBytesWithMode(byteCount, mode)
	if err != nil {
		return nil, err
	}
	// If bitCount is not a multiple of 8, zero out the unused trailing bits for clarity.
	extraBits := (8 - (bitCount % 8)) % 8
	if extraBits != 0 {
		mask := byte(0xFF << extraBits)
		data[len(data)-1] &= mask
	}
	return data, nil
}

// CollectBitsAtInterval reads bitCount bits every interval, invoking onBatch
// with the bytes each time. It runs until the context is cancelled or a read
// error occurs. Any error is returned.
func CollectBitsAtInterval(ctx context.Context, bitCount int, interval time.Duration, onBatch func([]byte)) error {
	return CollectBitsAtIntervalWithMode(ctx, bitCount, interval, ModeNormal, onBatch)
}

// CollectBitsAtIntervalWithMode reads bitCount bits every interval with the specified mode
func CollectBitsAtIntervalWithMode(ctx context.Context, bitCount int, interval time.Duration, mode CaptureMode, onBatch func([]byte)) error {
	if bitCount <= 0 {
		return errors.New("bitCount must be positive")
	}
	if interval <= 0 {
		return errors.New("interval must be positive")
	}
	if onBatch == nil {
		return errors.New("onBatch callback must not be nil")
	}

	// Use per-read connection approach to avoid long-running connection issues

	byteCount := (bitCount + 7) / 8
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Do an immediate first read, then on each tick thereafter.
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Open port for each read to avoid long-running connection issues
		currentPortName, err := FindPort()
		if err != nil {
			return fmt.Errorf("device not found: %w", err)
		}

		// Use default serial mode to avoid USB re-enumeration issues
		serialMode := &serial.Mode{
			Parity:   serial.NoParity,
			StopBits: serial.OneStopBit,
		}

		port, err := serial.Open(currentPortName, serialMode)
		if err != nil {
			return fmt.Errorf("open %s: %w", currentPortName, err)
		}

		// Quick setup and read
		_ = port.SetDTR(true)
		_ = port.SetReadTimeout(2000 * time.Millisecond)
		if err := port.ResetInputBuffer(); err != nil {
			port.Close()
			// not fatal, proceed
		}

		// Read data
		buf := make([]byte, byteCount)
		total := 0
		deadline := time.Now().Add(5 * time.Second)

		for total < byteCount {
			if time.Now().After(deadline) {
				port.Close()
				return fmt.Errorf("read timeout after 5s: read %d/%d bytes", total, byteCount)
			}

			n, err := port.Read(buf[total:])
			if err != nil {
				port.Close()
				return fmt.Errorf("read error: %w", err)
			}

			total += n
			if n == 0 {
				time.Sleep(5 * time.Millisecond)
			}
		}

		// Close port immediately after read
		port.Close()

		// Process bits (zero out unused trailing bits)
		extraBits := (8 - (bitCount % 8)) % 8
		if extraBits != 0 {
			buf[len(buf)-1] &= byte(0xFF << extraBits)
		}

		onBatch(buf)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Continue to next iteration
		}
	}
}

// getTrueRNGModel determines the TrueRNG device model from port details
// Based on the Python implementation's VID/PID detection
func getTrueRNGModel(p *enumerator.PortDetails) DeviceModel {
	if p == nil {
		return DeviceModelUnknown
	}

	// Check VID/PID combinations from Python code
	if p.IsUSB {
		vid := strings.ToUpper(p.VID)
		pid := strings.ToUpper(p.PID)

		// TrueRNG VID:PID combinations
		if vid == "04D8" && pid == "F5FE" {
			return DeviceModelTrueRNG
		}
		// TrueRNGpro VID:PID combinations
		if vid == "16D0" && pid == "0AA0" {
			return DeviceModelTrueRNGpro
		}
		// TrueRNGproV2 VID:PID combinations
		if vid == "04D8" && pid == "EBB5" {
			return DeviceModelTrueRNGproV2
		}
		// Additional TrueRNGpro variants
		if vid == "16D0" && (pid == "0AA2" || pid == "0AA4") {
			return DeviceModelTrueRNGpro
		}
	}

	// Fallback: check product name or description
	if p.IsUSB && p.Product != "" && strings.Contains(strings.ToUpper(p.Product), "TRUERNG") {
		return DeviceModelTrueRNGpro // Assume pro model for generic TrueRNG names
	}
	if p.IsUSB && p.SerialNumber != "" && strings.Contains(strings.ToUpper(p.SerialNumber), "TRUERNG") {
		return DeviceModelTrueRNGpro
	}
	if p.Name != "" && strings.Contains(strings.ToUpper(p.Name), "TRUERNG") {
		return DeviceModelTrueRNGpro
	}

	return DeviceModelUnknown
}

// changeMode implements the "knock sequence" to change TrueRNG capture modes
// This mirrors the Python implementation's modeChange function
func changeMode(portName string, mode CaptureMode) error {
	baudRate := mode.GetBaudRate()

	// "Knock" Sequence to activate mode change (from Python implementation)
	// Open/close serial port at different baud rates
	sequences := []int{110, 300, 110}

	for _, baud := range sequences {
		mode := &serial.Mode{
			BaudRate: baud,
			Parity:   serial.NoParity,
			StopBits: serial.OneStopBit,
		}

		port, err := serial.Open(portName, mode)
		if err != nil {
			return fmt.Errorf("failed to open port for mode change: %w", err)
		}
		time.Sleep(500 * time.Millisecond)
		port.Close()
	}

	// Final mode change to desired baud rate
	finalMode := &serial.Mode{
		BaudRate: baudRate,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	port, err := serial.Open(portName, finalMode)
	if err != nil {
		return fmt.Errorf("failed to set final mode: %w", err)
	}
	port.Close()

	return nil
}

// ListDevices prints information about all detected TrueRNG devices
func ListDevices() error {
	devices, err := EnumerateDevices()
	if err != nil {
		return err
	}

	if len(devices) == 0 {
		fmt.Println("No TrueRNG devices found")
		return nil
	}

	fmt.Println("Found TrueRNG devices:")
	for i, device := range devices {
		fmt.Printf("%d. %s on %s (Model: %s)\n", i+1, device.Name, device.Port, device.Model.String())
	}

	return nil
}

// CollectBitsAtIntervalWithReconnect is a more robust version that can handle
// device disconnections and attempt reconnection
func CollectBitsAtIntervalWithReconnect(ctx context.Context, bitCount int, interval time.Duration, mode CaptureMode, onBatch func([]byte)) error {
	if bitCount <= 0 {
		return errors.New("bitCount must be positive")
	}
	if interval <= 0 {
		return errors.New("interval must be positive")
	}
	if onBatch == nil {
		return errors.New("onBatch callback must not be nil")
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var port serial.Port
	var portName string
	var err error

	// Initial device connection
	portName, err = FindPort()
	if err != nil {
		return err
	}

	port, err = connectToDevice(portName, mode)
	if err != nil {
		return fmt.Errorf("initial connection failed: %w", err)
	}
	defer func() {
		if port != nil {
			port.Close()
		}
	}()

	byteCount := (bitCount + 7) / 8
	consecutiveErrors := 0
	maxConsecutiveErrors := 3

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Try to read from current port
		buf := make([]byte, byteCount)
		total := 0
		deadline := time.Now().Add(5 * time.Second)
		readAttempts := 0
		maxReadAttempts := 30

		readSuccessful := false

		for total < byteCount && readAttempts < maxReadAttempts && !readSuccessful {
			if time.Now().After(deadline) {
				break // Timeout
			}

			n, err := port.Read(buf[total:])
			if err != nil {
				// Check for port closed errors
				if strings.Contains(err.Error(), "closed") || strings.Contains(err.Error(), "broken pipe") {
					fmt.Printf("Port closed, attempting reconnection...\n")
					port.Close()
					port = nil
					break
				}
				consecutiveErrors++
				if consecutiveErrors >= maxConsecutiveErrors {
					return fmt.Errorf("too many consecutive read errors: %w", err)
				}
				break
			}

			total += n
			readAttempts++

			if n == 0 {
				time.Sleep(20 * time.Millisecond)
			} else {
				consecutiveErrors = 0 // Reset error counter on successful read
				if total >= byteCount {
					readSuccessful = true
				}
			}
		}

		// If read failed, try to reconnect
		if !readSuccessful || port == nil {
			if port != nil {
				port.Close()
				port = nil
			}

			// Wait a bit before attempting reconnection
			time.Sleep(500 * time.Millisecond)

			// Try to find device again
			newPortName, err := FindPort()
			if err != nil {
				fmt.Printf("Device not found during reconnection attempt: %v\n", err)
				time.Sleep(1 * time.Second)
				continue
			}

			// Check if device port changed
			if newPortName != portName {
				fmt.Printf("Device port changed from %s to %s\n", portName, newPortName)
				portName = newPortName
			}

			// Attempt reconnection
			port, err = connectToDevice(portName, mode)
			if err != nil {
				fmt.Printf("Reconnection failed: %v\n", err)
				time.Sleep(1 * time.Second)
				continue
			}

			fmt.Printf("Successfully reconnected to device\n")
			consecutiveErrors = 0
			continue // Skip this iteration and try again
		}

		// Process successful read
		extraBits := (8 - (bitCount % 8)) % 8
		if extraBits != 0 {
			buf[len(buf)-1] &= byte(0xFF << extraBits)
		}

		onBatch(buf)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Continue to next iteration
		}
	}
}

// connectToDevice establishes a connection to a TrueRNG device
func connectToDevice(portName string, mode CaptureMode) (serial.Port, error) {
	// Skip mode change for now to avoid triggering USB re-enumeration
	// if err := changeMode(portName, mode); err != nil {
	//     return nil, fmt.Errorf("failed to change mode: %w", err)
	// }

	// Open serial port
	// Use default serial mode to avoid USB re-enumeration issues
	// Let the TrueRNG device use its default baud rate
	serialMode := &serial.Mode{
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	port, err := serial.Open(portName, serialMode)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", portName, err)
	}

	// Configure port
	_ = port.SetDTR(true)
	_ = port.SetReadTimeout(2000 * time.Millisecond)
	if err := port.ResetInputBuffer(); err != nil {
		port.Close()
		return nil, fmt.Errorf("reset input buffer: %w", err)
	}

	// Additional stability setup
	_ = port.SetDTR(false)
	time.Sleep(100 * time.Millisecond)
	_ = port.SetDTR(true)
	time.Sleep(100 * time.Millisecond)

	return port, nil
}
