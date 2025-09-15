#!/bin/bash

# Unified Linux setup for BitBabbler and TrueRNG devices
# - Installs udev rules
# - Creates required groups (bit-babbler)
# - Reloads udev so devices are ready without reboot

set -e

echo "🔧 Setting up BitBabbler and TrueRNG device support for Linux..."
echo ""

# Must be root
if [[ $EUID -ne 0 ]]; then
  echo "❌ This script must be run as root (sudo)"
  exit 1
fi

########################################
# BitBabbler setup
########################################
echo "📦 Ensuring 'bit-babbler' system group exists..."
if ! getent group bit-babbler > /dev/null 2>&1; then
  groupadd --system bit-babbler
  echo "✅ Created bit-babbler group"
else
  echo "ℹ️  bit-babbler group already exists"
fi

echo ""
echo "🔧 Installing BitBabbler udev rules..."
BB_UDEV_DST="/etc/udev/rules.d/60-bit-babbler.rules"
BB_UDEV_SRC="/home/thiagojm/Projects/rng_cli_linux/bit-babbler-0.9/Makeup/config/acfile.60-bit-babbler.rules"
if [[ -f "$BB_UDEV_SRC" ]]; then
  cp "$BB_UDEV_SRC" "$BB_UDEV_DST"
  chmod 644 "$BB_UDEV_DST"
  echo "✅ Installed BitBabbler rules to $BB_UDEV_DST"
else
  echo "❌ BitBabbler vendor udev file not found at $BB_UDEV_SRC"
  exit 1
fi

echo ""
echo "⚙️  Setting up BitBabbler sysctl configuration..."
BB_SYSCTL_DST="/etc/sysctl.d/bit-babbler-sysctl.conf"
BB_SYSCTL_SRC="/home/thiagojm/Projects/rng_cli_linux/bit-babbler-0.9/Makeup/config/acfile.bit-babbler-sysctl.conf"
if [[ -f "$BB_SYSCTL_SRC" ]]; then
  cp "$BB_SYSCTL_SRC" "$BB_SYSCTL_DST"
  chmod 644 "$BB_SYSCTL_DST"
  echo "✅ Installed BitBabbler sysctl config to $BB_SYSCTL_DST"
  echo "🔄 Applying sysctl settings..."
  sysctl -q -p "$BB_SYSCTL_DST" || true
  echo "✅ Applied sysctl settings"
else
  echo "❌ BitBabbler sysctl file not found at $BB_SYSCTL_SRC"
  exit 1
fi

########################################
# TrueRNG setup
########################################
echo ""
echo "🔧 Installing TrueRNG udev rules..."
TRNG_UDEV_DST="/etc/udev/rules.d/99-TrueRNG.rules"
TRNG_UDEV_SRC="/home/thiagojm/Projects/rng_cli_linux/installers/truerng/udev_rules/99-TrueRNG.rules"
if [[ -f "$TRNG_UDEV_SRC" ]]; then
  cp "$TRNG_UDEV_SRC" "$TRNG_UDEV_DST"
  chmod 644 "$TRNG_UDEV_DST"
  echo "✅ Installed TrueRNG rules to $TRNG_UDEV_DST"
else
  echo "❌ TrueRNG udev file not found at $TRNG_UDEV_SRC"
  exit 1
fi

########################################
# Apply settings
########################################
echo ""
echo "🔄 Reloading udev rules & triggering..."
udevadm control --reload-rules
udevadm trigger
echo "✅ udev reloaded"

########################################
# Add invoking user to bit-babbler group
########################################
TARGET_USER="${SUDO_USER:-$USER}"
if [[ -n "$TARGET_USER" && "$TARGET_USER" != "root" ]]; then
  echo ""
  echo "👥 Ensuring user '$TARGET_USER' is in 'bit-babbler' group..."
  if id -nG "$TARGET_USER" | tr " " "\n" | grep -qx "bit-babbler"; then
    echo "ℹ️  $TARGET_USER already in bit-babbler group"
  else
    usermod -aG bit-babbler "$TARGET_USER"
    echo "✅ Added $TARGET_USER to bit-babbler group"
  fi
else
  echo ""
  echo "ℹ️  Skipping group membership update (no non-root invoking user detected)"
fi

########################################
# Optional driver checks
########################################
echo ""
echo "🔍 Checking for FTDI driver (ftdi_sio)..."
if lsmod | grep -q ftdi_sio; then
  echo "✅ FTDI serial driver is loaded"
else
  echo "⚠️  FTDI driver not currently loaded"
  echo "   You can load it now with: sudo modprobe ftdi_sio"
fi

echo ""
echo "🎉 Setup complete!"
echo ""
echo "📋 Next steps (recommended):"
echo "   • Log out/in (or run: exec su - ${SUDO_USER:-$USER}) to refresh group membership"
echo "   • Replug your devices or keep them plugged; udev rules have been triggered"
echo "   • TrueRNG rules grant MODE=0666; no additional groups usually required"


