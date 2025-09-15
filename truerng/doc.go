// Package truerng provides utilities to detect and read random data from a
// TrueRNG USB device presented as a serial port on Linux. It mirrors
// the behavior of the provided Python script `truerng.py` while exposing a
// Go-friendly API that is suitable for use in GUI applications.
//
// Features:
//   - Comprehensive device detection for TrueRNG, TrueRNGpro, and TrueRNGproV2
//   - Multiple capture modes with baud rate switching
//   - Device model identification
//   - Cross-platform serial communication
package truerng
