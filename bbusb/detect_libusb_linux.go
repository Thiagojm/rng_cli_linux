//go:build linux

package bbusb

import (
	"github.com/google/gousb"
)

// detectUSBViaLibusb checks for the BitBabbler device using libusb via gousb.
// It returns true if a device with the expected VID/PID is present.
func detectUSBViaLibusb() bool {
	ctx := gousb.NewContext()
	defer ctx.Close()

	dev, err := ctx.OpenDeviceWithVIDPID(gousb.ID(ftdiVendorID), gousb.ID(bbProductID))
	if err != nil {
		return false
	}
	if dev == nil {
		return false
	}
	_ = dev.Close()
	return true
}
