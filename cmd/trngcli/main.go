// trngcli is an enhanced CLI demonstrating usage of the truerng package.
// It supports device detection, mode selection, and reading at intervals.
package main

import (
	"context"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/Thiagojm/rng_cli_linux/truerng"
)

func main() {
	bits := flag.Int("bits", 1024, "number of bits to read per batch")
	interval := flag.Duration("interval", 0, "interval between reads (e.g. 2s). 0 for one-shot")
	modeStr := flag.String("mode", "normal", "(deprecated - now uses default serial configuration)")
	list := flag.Bool("list", false, "list all detected TrueRNG devices")
	reconnect := flag.Bool("reconnect", false, "enable automatic reconnection on device disconnection")
	flag.Parse()

	// List devices if requested
	if *list {
		if err := truerng.ListDevices(); err != nil {
			log.Fatalf("list devices error: %v", err)
		}
		return
	}

	// Parse capture mode
	var mode truerng.CaptureMode
	switch strings.ToLower(*modeStr) {
	case "normal":
		mode = truerng.ModeNormal
	case "psdebug":
		mode = truerng.ModePSDebug
	case "rngdebug":
		mode = truerng.ModeRNGDebug
	case "rng1white":
		mode = truerng.ModeRNG1White
	case "rng2white":
		mode = truerng.ModeRNG2White
	case "raw_bin":
		mode = truerng.ModeRawBin
	case "raw_asc":
		mode = truerng.ModeRawASC
	case "unwhitened":
		mode = truerng.ModeUnwhitened
	case "normal_asc":
		mode = truerng.ModeNormalASC
	case "normal_asc_slow":
		mode = truerng.ModeNormalASCSlow
	default:
		log.Fatalf("unknown mode: %s", *modeStr)
	}

	// Detect device and show info
	device, err := truerng.FindDevice()
	if err != nil {
		log.Fatalf("device detection error: %v", err)
	}

	fmt.Printf("Using TrueRNG device: %s on %s (Model: %s)\n",
		device.Name, device.Port, device.Model.String())
	fmt.Printf("Using default serial configuration (no mode switching)\n")

	if *interval == 0 {
		data, err := truerng.ReadBitsWithMode(*bits, mode)
		if err != nil {
			log.Fatalf("read error: %v", err)
		}
		fmt.Printf("read %d bits (%d bytes)\n", *bits, len(data))
		fmt.Printf("%s\n", hex.EncodeToString(data))
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if *reconnect {
		log.Printf("reading %d bits every %s with auto-reconnect. press Ctrl+C to stop...", *bits, interval.String())
		err = truerng.CollectBitsAtIntervalWithReconnect(ctx, *bits, *interval, mode, func(b []byte) {
			fmt.Printf("%s  %d bits  %s\n", time.Now().Format(time.RFC3339), *bits, hex.EncodeToString(b))
		})
	} else {
		log.Printf("reading %d bits every %s. press Ctrl+C to stop...", *bits, interval.String())
		err = truerng.CollectBitsAtIntervalWithMode(ctx, *bits, *interval, mode, func(b []byte) {
			fmt.Printf("%s  %d bits  %s\n", time.Now().Format(time.RFC3339), *bits, hex.EncodeToString(b))
		})
	}

	if err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("collect error: %v", err)
	}
}
