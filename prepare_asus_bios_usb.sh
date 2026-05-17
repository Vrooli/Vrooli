#!/usr/bin/env bash
set -euo pipefail

DEV="${1:-/dev/sda}"
PART="${DEV}1"
EXPECTED_SERIAL="90006C46B27D2752"
EXPECTED_ZIP_SHA="0f5cf0ea8012a1a5358d5c9c71ccc98b4ad6717787167a02f03214c066c5c8d1"
OWNER_USER="${SUDO_USER:-$(id -un)}"
OWNER_HOME="$(getent passwd "$OWNER_USER" | cut -d: -f6)"
ZIP="$OWNER_HOME/Downloads/ProArt-X870E-CREATOR-WIFI-ASUS-2202.zip"
SRC="$OWNER_HOME/Downloads/ProArt-X870E-CREATOR-WIFI-ASUS-2202/ProArt-X870E-CREATOR-WIFI-ASUS-2202.CAP"

die() {
  printf 'ERROR: %s\n' "$*" >&2
  exit 1
}

[[ "$(id -u)" -eq 0 ]] || die "Run this script with sudo."
[[ -b "$DEV" ]] || die "$DEV is not a block device."
[[ -f "$ZIP" ]] || die "BIOS zip not found: $ZIP"
[[ -f "$SRC" ]] || die "Extracted CAP not found: $SRC"

actual_serial="$(udevadm info --query=property --name="$DEV" | sed -n 's/^ID_SERIAL_SHORT=//p')"
actual_bus="$(udevadm info --query=property --name="$DEV" | sed -n 's/^ID_BUS=//p')"
actual_model="$(udevadm info --query=property --name="$DEV" | sed -n 's/^ID_MODEL=//p')"

[[ "$actual_bus" == "usb" ]] || die "$DEV is not reported as a USB device."
[[ "$actual_serial" == "$EXPECTED_SERIAL" ]] || die "$DEV serial is '$actual_serial', expected '$EXPECTED_SERIAL'."

zip_sha="$(sha256sum "$ZIP" | awk '{print $1}')"
[[ "$zip_sha" == "$EXPECTED_ZIP_SHA" ]] || die "BIOS zip SHA-256 does not match ASUS published hash."

printf 'Target device confirmed:\n'
lsblk -o NAME,PATH,SIZE,MODEL,SERIAL,TRAN,TYPE,FSTYPE,LABEL,MOUNTPOINTS "$DEV"
printf '\nModel: %s\nSerial: %s\n\n' "$actual_model" "$actual_serial"
printf 'This will ERASE %s and replace it with a FAT32 BIOS update USB.\n' "$DEV"
read -r -p "Type ERASE to continue: " confirmation
[[ "$confirmation" == "ERASE" ]] || die "Confirmation not received; aborting."

umount "${DEV}"* 2>/dev/null || true
wipefs -a "$DEV"
parted "$DEV" --script mklabel msdos
parted "$DEV" --script mkpart primary fat32 1MiB 100%
partprobe "$DEV"
udevadm settle
sleep 2

[[ -b "$PART" ]] || die "Expected partition was not created: $PART"
mkfs.vfat -F 32 -n BIOS "$PART"

mount_dir="$(mktemp -d /mnt/asus-bios-usb.XXXXXX)"
cleanup() {
  if mountpoint -q "$mount_dir"; then
    umount "$mount_dir"
  fi
  rmdir "$mount_dir" 2>/dev/null || true
}
trap cleanup EXIT

mount "$PART" "$mount_dir"
install -m 0644 "$SRC" "$mount_dir/A5560.CAP"
sync

src_sha="$(sha256sum "$SRC" | awk '{print $1}')"
dst_sha="$(sha256sum "$mount_dir/A5560.CAP" | awk '{print $1}')"
[[ "$src_sha" == "$dst_sha" ]] || die "Copied CAP hash mismatch."

printf '\nPrepared BIOS USB successfully:\n'
ls -lh "$mount_dir/A5560.CAP"
printf 'CAP SHA-256: %s\n' "$dst_sha"
printf 'USB label: BIOS\n'
printf 'BIOS filename: A5560.CAP\n'
