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
- **Data analysis**: Convert collected data to Excel format with statistical analysis and charts

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

### Quick setup (recommended)

Run the unified script to install rules for both devices, create groups, and apply settings:

```bash
sudo ./setup_rng_devices_linux.sh
# then refresh group membership in this shell (optional)
exec su - $USER
```

What this script does:
- Installs BitBabbler udev rules to `/etc/udev/rules.d/60-bit-babbler.rules`
- Installs BitBabbler sysctl config to `/etc/sysctl.d/bit-babbler-sysctl.conf` (applied immediately)
- Installs TrueRNG udev rules to `/etc/udev/rules.d/99-TrueRNG.rules`
- Ensures `bit-babbler` group exists and adds your user
- Reloads and triggers udev so rules take effect

Notes:
- Replug devices if they are not detected immediately after running the script
- The provided TrueRNG rules set `MODE=0666`, so `dialout` membership is not required
- If you prefer not to install the TrueRNG rules, add your user to `dialout` instead: `sudo usermod -aG dialout $USER`

### Manual setup (alternative)

If you want to configure devices separately:

- TrueRNG: either install the udev rules from `installers/truerng/udev_rules/99-TrueRNG.rules` (sets `MODE=0666`), or add your user to `dialout` and relogin.
- BitBabbler: run the vendor-based setup for udev rules and sysctl:
  ```bash
  sudo ./setup_bitbabbler_linux.sh
  sudo usermod -aG bit-babbler $USER
  # relogin or: exec su - $USER
  ```
  If detection still fails due to permissions, you can test with:
  ```bash
  sudo -E CGO_ENABLED=1 go run ./cmd/bbdetect
  ```

## Unified Collector CLI

The `collect` command mirrors the Windows `collect` tool and supports `pseudo`, `trng`, and `bitb`.

Build:
```bash
# Pseudorng/TrueRNG only
go build -o collect ./cmd/collect
# Include BitBabbler (requires CGO/libusb)
CGO_ENABLED=1 go build -o collect ./cmd/collect
```

Usage:
```bash
./collect -device pseudo -bits 2048 -interval 1 -outdir data
./collect -device trng   -bits 2048 -interval 1 -outdir data
CGO_ENABLED=1 ./collect -device bitb -bits 2048 -interval 1 -outdir data
```

Output:
- Writes raw bytes to `.bin` and a CSV line per sample with the count of ones
- Filenames are constructed via `naming` to include device, bit size, and interval

## Pseudorandom CLI

The `pseudocli` command provides a simple interface to test pseudorandom generation:

```bash
# Build the CLI
go build -o pseudocli ./cmd/pseudocli

# Generate 1024 random bits (one-shot)
./pseudocli -bits 1024

# Generate 512 bits every 2 seconds (continuous)
./pseudocli -bits 512 -interval 2s
```

## TrueRNG CLI

```bash
# Build the CLI
go build -o trngcli ./cmd/trngcli

# List all detected TrueRNG devices
./trngcli -list

# One-shot collection
./trngcli -bits 1024
```

## BitBabbler CLI (Linux)

```bash
# Build (requires CGO)
CGO_ENABLED=1 go build -o bbdetect ./cmd/bbdetect
CGO_ENABLED=1 go build -o bb ./cmd/bb

# Detect BitBabbler device
./bbdetect

# Read 1024 bits every second
./bb -bits 1024 -interval 1s
```

## Data Analysis CLI

The `filetoexcel` command converts collected data files to Excel format with statistical analysis:

```bash
# Build the CLI
go build -o filetoexcel ./cmd/filetoexcel

# Convert CSV data to Excel with z-score analysis
./filetoexcel data/20250915T222739_bitb_s2048_i1.csv

# Convert binary data to Excel with z-score analysis
./filetoexcel data/20250915T222739_bitb_s2048_i1.bin
```

Features:
- **Statistical analysis**: Calculates cumulative mean and z-scores for randomness testing
- **Excel export**: Creates Excel files with data tables and line charts
- **Multiple formats**: Supports both CSV and binary input files
- **Automatic parsing**: Extracts sampling interval and bit count from filenames
- **Time formatting**: Handles various timestamp formats for CSV files

The generated Excel files contain:
- Data table with samples/time, ones count, cumulative mean, and z-test values
- Line chart visualizing z-score trends over time
- Proper axis labels and titles based on the input data parameters

## API Usage

See the README sections above for examples of pseudorng, truerng, and bbusb usage.

## Project Structure

```
rng_cli_linux/
├── cmd/
│   ├── pseudocli/          # Pseudorandom CLI demo
│   ├── trngcli/            # TrueRNG CLI demo
│   ├── bb/                 # BitBabbler data collection CLI
│   ├── bbdetect/           # BitBabbler device detection CLI
│   ├── collect/            # Unified collector (pseudo|trng|bitb)
│   └── filetoexcel/        # Data analysis and Excel export CLI
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

## License

See LICENSE file for licensing information.
