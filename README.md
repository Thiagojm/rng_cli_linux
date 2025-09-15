# rng_cli_linux

Random data collection CLI in Go for Linux systems.

## Overview

This is a Linux port of the `rng_go_cli` project, originally designed for Windows. It provides random data collection from multiple sources:
- **Pseudorandom (software)**: High-quality software-based random generation
- **TrueRNG (hardware)**: USB-based true random number generator
- **BitBabbler (hardware)**: FTDI-based USB true-random device, using libusb on Linux

## Features

- **Pseudorandom generation**: Uses Go's `crypto/rand` and `math/rand` for high-quality pseudorandom data
- **TrueRNG hardware support**: USB-based true random number generator detection and reading
- **BitBabbler hardware support (Linux)**: Detection via libusb (gousb) and bulk-EP reads, with serial fallback where applicable
- **Flexible output**: Generate random bits in various sizes
- **Interval collection**: Collect data continuously at specified intervals
- **Deterministic generator**: Create reproducible random streams with seeded generators
- **Cross-platform serial support**: Automatic detection of TrueRNG devices on Linux serial ports

## Requirements

- Go 1.24.0 or later
- Linux operating system
- For BitBabbler (libusb path) build/run on Linux:
  - `libusb-1.0-0-dev`
  - `build-essential` (GCC, make)
  - `pkg-config`

Install build prerequisites on Debian/Ubuntu:
```bash
sudo apt-get update
sudo apt-get install -y libusb-1.0-0-dev build-essential pkg-config
```

## Installation

1. Clone or download the project
2. Navigate to the project directory
3. Initialize dependencies:
   ```bash
   go mod tidy
   ```

## Device Setup (Linux)

### TrueRNG
- Ensure your user has serial access: `sudo usermod -a -G dialout $USER`
- Replug device or log out/in after adding to the group

### BitBabbler
- Install udev rules and sysctl settings (provided in this repo):
  ```bash
  sudo ./setup_bitbabbler_linux.sh
  ```
- Add your user to the device group and replug:
  ```bash
  sudo usermod -aG bit-babbler $USER
  newgrp bit-babbler
  # unplug/replug the BitBabbler device
  ```
- If detection fails due to permissions, test with sudo:
  ```bash
  sudo -E CGO_ENABLED=1 go run ./cmd/bbdetect
  ```

## Usage

### Pseudorandom CLI

The `pseudocli` command provides a simple interface to test pseudorandom generation:

```bash
# Build the CLI
go build -o pseudocli ./cmd/pseudocli

# Generate 1024 random bits (one-shot)
./pseudocli -bits 1024

# Generate 512 bits every 2 seconds (continuous)
./pseudocli -bits 512 -interval 2s
```

#### Flags:
- `-bits` (int): Number of bits to read per batch (default: 1024)
- `-interval` (duration): Interval between reads (e.g., 2s, 500ms). Use 0 for one-shot mode (default: 0)

#### Examples:

```bash
# One-shot 2048 bits
go run ./cmd/pseudocli -bits 2048

# Continuous collection every 1 second
go run ./cmd/pseudocli -bits 1024 -interval 1s
# Press Ctrl+C to stop continuous collection
```

### TrueRNG CLI

The `trngcli` command works with TrueRNG USB hardware devices with enhanced features:

```bash
# Build the CLI
go build -o trngcli ./cmd/trngcli

# List all detected TrueRNG devices
./trngcli -list

# Generate 1024 true random bits (one-shot)
./trngcli -bits 1024

# Generate with different capture modes
./trngcli -bits 1024 -mode raw_bin
./trngcli -bits 1024 -mode unwhitened

# Generate 512 true random bits every 2 seconds (continuous)
./trngcli -bits 512 -interval 2s
```

#### TrueRNG Setup:
- Connect your TrueRNG USB device
- Ensure your user has serial port access: `sudo usermod -a -G dialout $USER`
- The CLI automatically detects TrueRNG, TrueRNGpro, and TrueRNGproV2 devices

#### Supported Capture Modes:
- `normal` - Combined streams + Mersenne Twister (default, 300 baud)
- `raw_bin` - Raw ADC samples in binary (19200 baud)
- `raw_asc` - Raw ADC samples in ASCII (38400 baud)
- `unwhitened` - Unwhitened RNG1-RNG2 (57600 baud, TrueRNGproV2 only)
- `psdebug`, `rngdebug`, `rng1white`, `rng2white`, `normal_asc`, `normal_asc_slow`

#### Examples:

```bash
# List available devices
go run ./cmd/trngcli -list

# One-shot 2048 true random bits
go run ./cmd/trngcli -bits 2048

# Read raw ADC samples
go run ./cmd/trngcli -bits 2048 -mode raw_bin

# Continuous collection every 1 second with specific mode
go run ./cmd/trngcli -bits 1024 -interval 1s -mode unwhitened
# Press Ctrl+C to stop continuous collection
```

### BitBabbler CLI (Linux)

BitBabbler uses libusb for detection and bulk reads on Linux.

```bash
# Build the CLIs with CGO enabled (required for libusb)
CGO_ENABLED=1 go build -o bbdetect ./cmd/bbdetect
CGO_ENABLED=1 go build -o bb ./cmd/bb

# Detect BitBabbler device
./bbdetect

# Read 1024 bits every second
./bb -bits 1024 -interval 1s
```

#### Notes
- If detection works only with sudo, it’s a permission issue. Ensure you’ve run `setup_bitbabbler_linux.sh` and added your user to `bit-babbler` group, then replug the device.

## API Usage

#### Basic pseudorandom generation:
```go
import "github.com/Thiagojm/rng_cli_linux/pseudorng"

// Generate random bits
data, err := pseudorng.ReadBits(2048)
if err != nil {
    log.Fatal(err)
}
```

#### Continuous collection:
```go
import (
    "context"
    "time"
    "github.com/Thiagojm/rng_cli_linux/pseudorng"
)

ctx := context.Background()
err := pseudorng.CollectBitsAtInterval(ctx, 1024, 1*time.Second, func(batch []byte) {
    // Process each batch
    fmt.Printf("Received %d bytes\n", len(batch))
})
```

#### Deterministic generator:
```go
// Create a seeded generator for reproducible results
gen, err := pseudorng.NewGenerator(12345)
if err != nil {
    log.Fatal(err)
}

data, err := gen.ReadBits(512)
// Use the same seed to get identical results
```

#### TrueRNG hardware access:
```go
import "github.com/Thiagojm/rng_cli_linux/truerng"

// Detect TrueRNG device
present, err := truerng.Detect()
if err != nil {
    log.Fatal(err)
}
if !present {
    log.Fatal("TrueRNG device not found")
}

// Get detailed device information
device, err := truerng.FindDevice()
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Found %s on %s\n", device.Model.String(), device.Port)
```

#### BitBabbler hardware access (Linux):
```go
import "github.com/Thiagojm/rng_cli_linux/bbusb"

// Detect BitBabbler device (VID 0x0403, PID 0x7840)
present, err := bbusb.Detect()
if err != nil {
    log.Fatal(err)
}
if !present {
    log.Fatal("BitBabbler device not found")
}

// Get detailed device information
device, err := bbusb.FindDevice()
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Found %s (%s)\n", device.FriendlyName, device.DevicePath)

// Open device and read random data
session, err := bbusb.OpenBitBabbler(2500000, 1)
if err != nil {
    log.Fatal(err)
}
defer session.Close()

// Read random bytes
buf := make([]byte, 1024)
n, err := session.ReadRandom(buf)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Read %d bytes of random data\n", n)
```

## Project Structure

```
rng_cli_linux/
├── cmd/
│   ├── pseudocli/          # Pseudorandom CLI demo
│   ├── trngcli/            # TrueRNG CLI demo
│   ├── bb/                 # BitBabbler data collection CLI
│   └── bbdetect/           # BitBabbler device detection CLI
├── pseudorng/              # Pseudorandom number generation package
├── truerng/                # TrueRNG hardware access package
├── bbusb/                  # BitBabbler hardware access package
├── naming/                 # Filename convention helpers
├── data/                   # Output directory for generated files
├── go.mod                  # Go module definition
└── README.md              # This file
```

## Differences from Windows Version

- **Cross-platform serial support**: Uses `go.bug.st/serial` for Linux serial port access
- **Linux device paths**: Detects TrueRNG devices on `/dev/ttyUSB*` and `/dev/ttyACM*` paths
- **BitBabbler (Linux)**: Uses libusb (gousb) to detect/read via bulk endpoints. No long-running daemon is used.
- **Linux permissions**: Includes setup instructions for proper udev rules and group permissions
- **Module name**: Updated to `github.com/Thiagojm/rng_cli_linux`
- **Build process**: CGO required for BitBabbler libusb path

## Future Enhancements

- Additional CLI commands for data analysis
- File output capabilities (.bin and .csv formats)
- Statistical analysis tools for randomness testing
- GUI applications for data collection

## License

See LICENSE file for licensing information.
