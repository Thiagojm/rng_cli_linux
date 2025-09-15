//go:build linux

package bbusb

import (
	"fmt"

	"github.com/google/gousb"
)

// FindDevice (Linux) uses libusb first, then falls back to serial enumeration.
func FindDevice() (*DeviceInfo, error) {
	// libusb path
	ctx := gousb.NewContext()
	defer ctx.Close()

	dev, err := ctx.OpenDeviceWithVIDPID(gousb.ID(ftdiVendorID), gousb.ID(bbProductID))
	if err == nil && dev != nil {
		name := ""
		_ = dev.Close()
		return &DeviceInfo{
			DevicePath:   fmt.Sprintf("usb:%04x:%04x", ftdiVendorID, bbProductID),
			HardwareIDs:  []string{fmt.Sprintf("USB\\VID_%04X&PID_%04X", ftdiVendorID, bbProductID)},
			FriendlyName: name,
		}, nil
	}

	// Fallback to the non-Linux impl (serial enumeration)
	return findDeviceSerialFallback()
}

// EnumerateDevices (Linux) via libusb with serial fallback for richer info.
func EnumerateDevices() ([]DeviceInfo, error) {
	var out []DeviceInfo
	ctx := gousb.NewContext()
	defer ctx.Close()

	devs, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		return desc.Vendor == gousb.ID(ftdiVendorID) && desc.Product == gousb.ID(bbProductID)
	})
	if err == nil {
		for _, d := range devs {
			name := ""
			out = append(out, DeviceInfo{
				DevicePath:   fmt.Sprintf("usb:%04x:%04x", ftdiVendorID, bbProductID),
				HardwareIDs:  []string{fmt.Sprintf("USB\\VID_%04X&PID_%04X", ftdiVendorID, bbProductID)},
				FriendlyName: name,
			})
			_ = d.Close()
		}
		if len(out) > 0 {
			return out, nil
		}
	}

	// Fallback
	return enumerateDevicesSerialFallback()
}
