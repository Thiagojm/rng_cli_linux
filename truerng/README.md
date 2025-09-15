## truerng package

Enhanced utilities to detect and read random data from TrueRNG USB devices exposed as serial ports on Linux. Features comprehensive device detection, multiple capture modes, and improved compatibility with the Python reference implementation.

### Import

```go
import "github.com/Thiagojm/rng_cli_linux/truerng"
```

### Device Detection and Enumeration

```go
// Simple detection
present, err := truerng.Detect()
if err != nil { /* handle */ }
if !present { /* handle not found */ }

// Get detailed device information
device, err := truerng.FindDevice()
if err != nil { /* handle */ }
fmt.Printf("Found %s on %s\n", device.Model.String(), device.Port)

// List all detected devices
devices, err := truerng.EnumerateDevices()
for _, dev := range devices {
    fmt.Printf("%s: %s on %s\n", dev.Model.String(), dev.Name, dev.Port)
}
```

### Reading with Capture Modes

TrueRNG devices support multiple capture modes with different characteristics:

```go
// Available modes
mode := truerng.ModeNormal        // Default: 300 baud, whitened output
mode := truerng.ModeRawBin        // 19200 baud, raw ADC samples (binary)
mode := truerng.ModeUnwhitened    // 57600 baud, unwhitened RNG1-RNG2 (TrueRNGproV2 only)

// Read bytes with specific mode
data, err := truerng.ReadBytesWithMode(64, mode)

// Read bits with specific mode
bits, err := truerng.ReadBitsWithMode(2048, mode)
```

### Supported Capture Modes

| Mode | Baud Rate | Description |
|------|-----------|-------------|
| `ModeNormal` | 300 | Combined streams + Mersenne Twister (default) |
| `ModePSDebug` | 1200 | Power supply voltage in mV (ASCII) |
| `ModeRNGDebug` | 2400 | RNG debug output (ASCII) |
| `ModeRNG1White` | 4800 | RNG1 + Mersenne Twister |
| `ModeRNG2White` | 9600 | RNG2 + Mersenne Twister |
| `ModeRawBin` | 19200 | Raw ADC samples (binary) |
| `ModeRawASC` | 38400 | Raw ADC samples (ASCII) |
| `ModeUnwhitened` | 57600 | Unwhitened RNG1-RNG2 (TrueRNGproV2 only) |
| `ModeNormalASC` | 115200 | Normal mode in ASCII (TrueRNGproV2 only) |
| `ModeNormalASCSlow` | 230400 | Normal mode ASCII - slow for small devices |

### Reading at Intervals

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// With default mode
err := truerng.CollectBitsAtInterval(ctx, 4096, 2*time.Second, func(b []byte) {
    // consume 4096 bits (packed in bytes)
})

// With specific mode
err := truerng.CollectBitsAtIntervalWithMode(ctx, 4096, 2*time.Second, truerng.ModeRawBin, func(b []byte) {
    // consume raw ADC samples
})
```

### Device Model Detection

The package automatically detects different TrueRNG device models:

- **TrueRNG**: VID `04D8`, PID `F5FE`
- **TrueRNGpro**: VID `16D0`, PIDs `0AA0`, `0AA2`, `0AA4`
- **TrueRNGproV2**: VID `04D8`, PID `EBB5`

```go
device, err := truerng.FindDevice()
switch device.Model {
case truerng.DeviceModelTrueRNG:
    fmt.Println("Basic TrueRNG detected")
case truerng.DeviceModelTrueRNGpro:
    fmt.Println("TrueRNGpro detected - supports all modes")
case truerng.DeviceModelTrueRNGproV2:
    fmt.Println("TrueRNGproV2 detected - supports ASCII modes")
}
```

### Behavior and Implementation Notes

- **Detection**: Uses VID/PID matching (primary) and product name matching (fallback)
- **Mode Switching**: Implements Python-style "knock sequence" for baud rate changes
- **Serial Communication**: Uses cross-platform `go.bug.st/serial` library
- **Timeout Handling**: 10-second read deadline prevents indefinite blocking
- **Bit Packing**: MSB-first within bytes, unused trailing bits zeroed
- **Error Recovery**: Mode change failures don't prevent reading in normal mode

### Linux Setup and Permissions

```bash
# Add user to dialout group for serial port access
sudo usermod -a -G dialout $USER

# Log out and back in for group changes to take effect
# Or use newgrp: newgrp dialout

# Check device permissions
ls -la /dev/ttyUSB* /dev/ttyACM*
```

### CLI Usage Examples

```bash
# List all detected devices
./trngcli -list

# Read 1024 bits in normal mode (default)
./trngcli -bits 1024

# Read with raw binary mode
./trngcli -bits 1024 -mode raw_bin

# Continuous reading every 2 seconds
./trngcli -bits 1024 -interval 2s

# Continuous reading with specific mode
./trngcli -bits 1024 -interval 2s -mode unwhitened
```

### Compatibility with Python Implementation

This Go implementation mirrors the behavior of `truerng_read.py`:
- Same VID/PID detection logic
- Identical mode change "knock sequence"
- Compatible serial port handling
- Similar timeout and error handling
- Equivalent device enumeration approach
