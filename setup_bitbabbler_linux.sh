#!/bin/bash

# BitBabbler Linux Setup Script
# Based on vendor's Debian package installation

set -e

echo "🔧 Setting up BitBabbler device support for Linux..."
echo ""

# Check if running as root
if [[ $EUID -ne 0 ]]; then
   echo "❌ This script must be run as root (sudo)"
   exit 1
fi

# Create bit-babbler group
echo "📦 Creating bit-babbler group..."
if ! getent group bit-babbler > /dev/null 2>&1; then
    groupadd --system bit-babbler
    echo "✅ Created bit-babbler group"
else
    echo "ℹ️  bit-babbler group already exists"
fi

# Install udev rules
echo ""
echo "🔧 Installing udev rules..."
UDEV_RULES_FILE="/etc/udev/rules.d/60-bit-babbler.rules"
VENDOR_RULES_FILE="/home/thiagojm/Projects/rng_cli_linux/bit-babbler-0.9/Makeup/config/acfile.60-bit-babbler.rules"

if [[ -f "$VENDOR_RULES_FILE" ]]; then
    cp "$VENDOR_RULES_FILE" "$UDEV_RULES_FILE"
    chmod 644 "$UDEV_RULES_FILE"
    echo "✅ Installed udev rules to $UDEV_RULES_FILE"
else
    echo "❌ Vendor udev rules file not found at $VENDOR_RULES_FILE"
    exit 1
fi

# Install sysctl configuration
echo ""
echo "⚙️  Setting up sysctl configuration..."
SYSCTL_FILE="/etc/sysctl.d/bit-babbler-sysctl.conf"
VENDOR_SYSCTL_FILE="/home/thiagojm/Projects/rng_cli_linux/bit-babbler-0.9/Makeup/config/acfile.bit-babbler-sysctl.conf"

if [[ -f "$VENDOR_SYSCTL_FILE" ]]; then
    cp "$VENDOR_SYSCTL_FILE" "$SYSCTL_FILE"
    chmod 644 "$SYSCTL_FILE"
    echo "✅ Installed sysctl config to $SYSCTL_FILE"
else
    echo "❌ Vendor sysctl file not found at $VENDOR_SYSCTL_FILE"
    exit 1
fi

# Apply sysctl settings immediately
echo ""
echo "🔄 Applying sysctl settings..."
sysctl -q -p "$SYSCTL_FILE" || true
echo "✅ Applied sysctl settings (kernel.random.write_wakeup_threshold = 2048)"

# Reload udev rules
echo ""
echo "🔄 Reloading udev rules..."
udevadm control --reload-rules
udevadm trigger
echo "✅ Reloaded udev rules"

# Check for FTDI drivers
echo ""
echo "🔍 Checking for FTDI drivers..."
if lsmod | grep -q ftdi_sio; then
    echo "✅ FTDI serial driver is loaded"
else
    echo "⚠️  FTDI serial driver not loaded - may need to install ftdi drivers"
    echo "   Try: sudo apt-get install linux-modules-extra-$(uname -r)"
fi

echo ""
echo "🎉 BitBabbler setup complete!"
echo ""
echo "📋 What was installed:"
echo "   • Created bit-babbler system group"
echo "   • Installed udev rules: $UDEV_RULES_FILE"
echo "   • Installed sysctl config: $SYSCTL_FILE"
echo "   • Set kernel.random.write_wakeup_threshold = 2048"
echo ""
echo "🔌 Next steps:"
echo "   1. Connect your BitBabbler device"
echo "   2. Check if it's detected: go run ./cmd/bbdetect"
echo "   3. If still not detected, you may need to install FTDI drivers:"
echo "      sudo apt-get install linux-modules-extra-$(uname -r)"
echo "   4. Or try loading the FTDI module manually:"
echo "      sudo modprobe ftdi_sio"
echo ""
echo "💡 If you still have issues, check dmesg for USB device messages:"
echo "   dmesg | grep -i usb"
echo "   dmesg | grep -i ftdi"
