//go:build !linux

package bbusb

import (
	"errors"
	"fmt"

	"go.bug.st/serial/enumerator"
)

func FindDevice() (*DeviceInfo, error) {
	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return nil, fmt.Errorf("enumerating ports: %w", err)
	}
	for _, p := range ports {
		if p == nil {
			continue
		}
		if hasBitBabblerVIDPID(p) {
			return &DeviceInfo{
				DevicePath:   p.Name,
				HardwareIDs:  []string{fmt.Sprintf("USB\\VID_%04X&PID_%04X", ftdiVendorID, bbProductID)},
				FriendlyName: p.Product,
			}, nil
		}
	}
	return nil, errors.New("BitBabbler device not found")
}

func EnumerateDevices() ([]DeviceInfo, error) {
	var devices []DeviceInfo
	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return nil, fmt.Errorf("enumerating ports: %w", err)
	}
	for _, p := range ports {
		if p == nil {
			continue
		}
		if hasBitBabblerVIDPID(p) {
			devices = append(devices, DeviceInfo{
				DevicePath:   p.Name,
				HardwareIDs:  []string{fmt.Sprintf("USB\\VID_%04X&PID_%04X", ftdiVendorID, bbProductID)},
				FriendlyName: p.Product,
			})
		}
	}
	return devices, nil
}
