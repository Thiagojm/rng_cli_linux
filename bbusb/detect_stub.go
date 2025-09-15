//go:build !linux

package bbusb

// Non-Linux platforms or when libusb detection isn't available.
func detectUSBViaLibusb() bool { return false }
