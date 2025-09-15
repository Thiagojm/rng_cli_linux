package main

import (
	"fmt"
	"log"

	"github.com/Thiagojm/rng_cli_linux/bbusb"
)

func main() {
	fmt.Println("BitBabbler Device Detection")
	fmt.Println("===========================")

	// Check if device is present
	present, err := bbusb.Detect()
	if err != nil {
		log.Fatalf("detection error: %v", err)
	}

	if !present {
		fmt.Println("❌ No BitBabbler device found")
		fmt.Println("\nMake sure your BitBabbler device is connected and powered on.")
		fmt.Println("BitBabbler devices use VID 0x0403 and PID 0x7840.")
		return
	}

	fmt.Println("✅ BitBabbler device detected!")

	// Get detailed device information
	device, err := bbusb.FindDevice()
	if err != nil {
		log.Printf("failed to get device info: %v", err)
		return
	}

	fmt.Println("\nDevice Information:")
	fmt.Printf("  Friendly Name: %s\n", device.FriendlyName)
	fmt.Printf("  Device Path: %s\n", device.DevicePath)
	fmt.Printf("  Hardware IDs: %v\n", device.HardwareIDs)

	// Try to enumerate all devices
	devices, err := bbusb.EnumerateDevices()
	if err != nil {
		log.Printf("failed to enumerate devices: %v", err)
		return
	}

	if len(devices) > 1 {
		fmt.Printf("\nFound %d BitBabbler devices:\n", len(devices))
		for i, dev := range devices {
			fmt.Printf("  %d. %s (%s)\n", i+1, dev.FriendlyName, dev.DevicePath)
		}
	}

	fmt.Println("\n🎉 Device is ready for use!")
	fmt.Println("You can now run: go run ./cmd/bb -bits 1024")
}
