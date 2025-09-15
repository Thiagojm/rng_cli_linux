package bbusb

import (
	"fmt"
	"strings"

	"go.bug.st/serial/enumerator"
)

// FTDI vendor/product for BitBabbler
// Based on vendor's specification: FTDI VID with BitBabbler-specific PID
const (
	ftdiVendorID = 0x0403 // FTDI Vendor ID
	bbProductID  = 0x7840 // BitBabbler Product ID
)

// mpsse constants mirrors
const (
	mpsseNoClkDiv5     = 0x8A
	mpsseNoAdaptiveClk = 0x97
	mpsseNo3PhaseClk   = 0x8D
	mpsseSetDataLow    = 0x80
	mpsseSetDataHigh   = 0x82
	mpsseSetClkDivisor = 0x86
	mpsseSendImmediate = 0x87

	// read bytes in, MSB first, sample on +ve edge (matches default vendor code path)
	mpsseDataByteInPosMSB = 0x20
)

// ftdi SIO requests (vendor-specific)
const (
	ftdiReqReset        = 0x00
	ftdiReqSetFlowCtrl  = 0x02
	ftdiReqSetBaudRate  = 0x03
	ftdiReqSetData      = 0x04
	ftdiReqGetModemStat = 0x05
	ftdiReqSetEventChar = 0x06
	ftdiReqSetErrorChar = 0x07
	ftdiReqSetLatency   = 0x09
	ftdiReqGetLatency   = 0x0A
	ftdiReqSetBitmode   = 0x0B
)

// ftdi reset values
const (
	ftdiResetSIO     = 0
	ftdiResetPurgeRX = 1
	ftdiResetPurgeTX = 2
)

// ftdi flow control
const (
	ftdiFlowNone   = 0x0000
	ftdiFlowRtsCts = 0x0100
)

// ftdi bitmodes
const (
	ftdiBitmodeReset = 0x0000
	ftdiBitmodeMpsse = 0x0200
)

// DeviceInfo contains key metadata for a detected BitBabbler device.
type DeviceInfo struct {
	// DevicePath is the system path to the device interface
	DevicePath string
	// HardwareIDs is the list of hardware IDs from the device
	HardwareIDs []string
	// FriendlyName is a human-friendly device label if present
	FriendlyName string
}

// Detect checks if a BitBabbler device (VID 0x0403, PID 0x7840) is present.
// Uses serial port enumeration to find FTDI devices with BitBabbler characteristics.
func Detect() (bool, error) {
	// Prefer libusb detection if available
	if detectUSBViaLibusb() {
		return true, nil
	}

	// Fallback to serial enumeration
	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return false, fmt.Errorf("enumerating ports: %w", err)
	}

	for _, p := range ports {
		if p == nil {
			continue
		}
		if hasBitBabblerVIDPID(p) {
			return true, nil
		}
	}
	return false, nil
}

// hasBitBabblerVIDPID checks if a port has BitBabbler VID/PID
// Based on vendor's code: VID 0x0403 (FTDI), PID 0x7840 (BitBabbler)
func hasBitBabblerVIDPID(p *enumerator.PortDetails) bool {
	if p == nil {
		return false
	}

	// Primary check: VID/PID from vendor's specification
	if p.IsUSB {
		vid := strings.ToUpper(p.VID)
		pid := strings.ToUpper(p.PID)

		// BitBabbler uses FTDI VID with specific PID
		if vid == "0403" && pid == "7840" {
			return true
		}
	}

	// Secondary check: Product strings and serial from vendor's code
	if p.IsUSB {
		// Check product name - vendor's code shows "BitBabbler"
		if p.Product != "" {
			productUpper := strings.ToUpper(p.Product)
			if strings.Contains(productUpper, "BITBABBLER") ||
				strings.Contains(productUpper, "BIT BABBLER") ||
				strings.Contains(productUpper, "BB ") {
				return true
			}
		}

		// Check serial number - vendor's code uses format like "BB000001"
		if p.SerialNumber != "" {
			serialUpper := strings.ToUpper(p.SerialNumber)
			if strings.HasPrefix(serialUpper, "BB") ||
				strings.Contains(serialUpper, "BITBABBLER") {
				return true
			}
		}
	}

	return false
}
