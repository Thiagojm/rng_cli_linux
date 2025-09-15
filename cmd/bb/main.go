package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/Thiagojm/rng_cli_linux/bbusb"
)

func main() {
	bits := flag.Int("bits", 1024, "number of bits to read per batch")
	bitrate := flag.Uint("bitrate", 2500000, "bitrate for BitBabbler (default 2.5M)")
	latency := flag.Uint("latency", 1, "FTDI latency timer in ms")
	flag.Parse()

	// Check if device is present
	present, err := bbusb.Detect()
	if err != nil {
		log.Fatalf("detection error: %v", err)
	}
	if !present {
		log.Fatal("BitBabbler device not found")
	}

	// Get device info
	device, err := bbusb.FindDevice()
	if err != nil {
		log.Fatalf("device info error: %v", err)
	}

	fmt.Printf("Found BitBabbler device: %s\n", device.FriendlyName)
	fmt.Printf("Device path: %s\n", device.DevicePath)
	fmt.Printf("Using serial mode (simplified - not full MPSSE)\n")

	// Open device session
	session, err := bbusb.OpenBitBabbler(*bitrate, uint8(*latency))
	if err != nil {
		log.Fatalf("failed to open BitBabbler: %v", err)
	}
	defer session.Close()

	fmt.Printf("BitBabbler device initialized successfully!\n")

	// Calculate byte count
	byteCount := (*bits + 7) / 8

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	log.Printf("reading %d bits (%d bytes) continuously. press Ctrl+C to stop...", *bits, byteCount)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		buf := make([]byte, byteCount)
		n, err := session.ReadRandom(buf)
		if err != nil {
			log.Printf("read error: %v", err)
			continue
		}

		// Process bits (zero out unused trailing bits)
		extraBits := (8 - (*bits % 8)) % 8
		if extraBits != 0 {
			buf[len(buf)-1] &= byte(0xFF << extraBits)
		}

		fmt.Printf("%s  %d bits  %s\n", time.Now().Format(time.RFC3339), *bits, hex.EncodeToString(buf[:n]))

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Continue to next iteration
		}
	}
}
