#!/usr/bin/env bash
# Exact Noble MT6639 USB Bluetooth preparation in a private working directory.
set -Eeuo pipefail
trap 'printf "Bluetooth recipe stopped at line %s\n" "$LINENO" >&2' ERR
umask 077
phase="$1"
abi="$2"
shift 2
work=$PWD
headers="/lib/modules/$abi/build"
source="$work/ubuntu-source"
subtree=drivers/bluetooth
modules=(btmtk btusb)

test "$(id -u)" -ne 0
test "$abi" = 7.0.0-30-generic
test "$(uname -r)" = "$abi"

build() {
    local out="$1" module
    local disabled=()
    mkdir "$out"
    mapfile -t disabled < <(
        # Match the literal Kbuild CONFIG expansion syntax.
        # shellcheck disable=SC2016
        sed -n 's/^obj-$(\(CONFIG_[A-Za-z0-9_]*\)).*/\1=n/p' "$source/$subtree/Makefile" |
            sed '/^CONFIG_BT_HCIBTUSB=/d; /^CONFIG_BT_MTK=/d'
    )
    make -j2 -C "$headers" "M=$source/$subtree" CC=x86_64-linux-gnu-gcc-13 "${disabled[@]}" modules > "$out/build.log" 2>&1 ||
        { tail -60 "$out/build.log" >&2; return 1; }
    diff -u <(printf '%s.o\n' "${modules[@]}") <(sed 's@.*/@@' "$source/$subtree/modules.order" | sort)
    cp "$source/$subtree/Module.symvers" "$source/$subtree/modules.order" "$out/"
    for module in "${modules[@]}"; do
        cp "$source/$subtree/$module.ko" "$out/"
        modprobe --show-modversions "$out/$module.ko" > "$out/$module.modversions"
        modinfo "$out/$module.ko" > "$out/$module.modinfo"
    done
}

if [[ "$phase" == stock ]]; then
    cmp "$headers/.config" "/boot/config-$abi"
    sha256sum "$headers/Module.symvers" "$headers/.config" "/boot/config-$abi" > kernel-inputs.sha256
    mkdir "$source"
    {
        tar -xzf inputs/linux-hwe-7.0_7.0.0.orig.tar.gz --strip-components=1 -C "$source" "linux-7.0/$subtree"
        gzip -cd inputs/linux-hwe-7.0_7.0.0-30.30~24.04.1.diff.gz |
            git -C "$source" apply -p1 --include="$subtree/**"
        (cd "$source" && find "$subtree" -type f -print0 | sort -z | xargs -0 sha256sum) > stock-source.sha256
        sha256sum --check <<< "3a3f4f5c326de822d28e6db84bd1db8e874750ef9836d555dfa5370a55d0d40d  stock-source.sha256"
    } > unpack.log
    build stock
    for module in "${modules[@]}"; do
        installed=$(modinfo -k "$abi" -F filename "$module")
        diff -u <(modprobe --show-modversions "$installed" | sort) <(sort "stock/$module.modversions")
        test "$(modinfo -F vermagic "stock/$module.ko")" = "$(modinfo -F vermagic "$installed")"
    done
    exit
fi

test "$phase" = candidate
test "$#" -gt 0
sha256sum --check kernel-inputs.sha256
git -C "$source" apply --check "$work/adapted.patch"
git -C "$source" apply "$work/adapted.patch"
build candidate
sha256sum --check kernel-inputs.sha256
cp kernel-inputs.sha256 unpack.log stock-source.sha256 candidate/
cp stock/Module.symvers candidate/stock.Module.symvers
cp stock/build.log candidate/stock-build.log
firmware='mediatek/mt7927/BT_RAM_CODE_MT6639_2_1_hdr.bin'
for module in "${modules[@]}"; do
    test "$(modinfo -F vermagic "candidate/$module.ko")" = "$(modinfo -F vermagic "stock/$module.ko")"
    test -z "$(modinfo -F signer "candidate/$module.ko")"
done
modinfo -F firmware stock/btmtk.ko stock/btusb.ko | sort -u > candidate/firmware-stock.txt
modinfo -F firmware candidate/btmtk.ko candidate/btusb.ko | sort -u > candidate/firmware-candidate.txt
test "$(comm -13 candidate/firmware-stock.txt candidate/firmware-candidate.txt)" = "$firmware"
test -z "$(comm -23 candidate/firmware-stock.txt candidate/firmware-candidate.txt)"

image="$work/copied-tree"
tree="$image/lib/modules/$abi"
overlay="$tree/updates/vrooli-mt6639"
mkdir -p "$image/lib/modules"
cp -a --reflink=auto "/lib/modules/$abi" "$image/lib/modules/"
test -d "$tree" && test ! -L "$tree"
unlink "$tree/build"
while IFS= read -r -d '' link; do
    target=$(readlink -f "$link")
    [[ "$target" == "$tree/"* ]]
done < <(find "$tree" -type l -print0)
cp /etc/depmod.d/ubuntu.conf "$work/depmod.conf"

index_tree() {
    local stage="$1"
    depmod -b "$image" -C "$work/depmod.conf" -a -e -E "$headers/Module.symvers" "$abi" > "candidate/depmod-$stage.log" 2>&1
    if test -s "candidate/depmod-$stage.log"; then
        tail -60 "candidate/depmod-$stage.log" >&2
        return 1
    fi
    cp "$tree/modules.dep" "candidate/modules-$stage.dep"
}
lookup() {
    local alias module
    for module in "${modules[@]}"; do
        modinfo -b "$image" -k "$abi" -F filename "$module"
    done
    for alias; do
        test "$(modprobe -C /dev/null -d "$image" -S "$abi" --resolve-alias "$alias")" = btusb
        modprobe -C /dev/null -d "$image" -S "$abi" --show-depends "$alias"
    done
}
hash_tree() {
    (cd "$tree" && find . -type f -print0 | sort -z | xargs -0 -r sha256sum) > "candidate/tree-$1.sha256"
    (cd "$tree" && find . -type l -printf '%p -> %l\n' | sort) > "candidate/tree-$1.links"
}

index_tree stock
lookup "$@" > candidate/lookup-stock.log
hash_tree stock
test ! -L "$(dirname "$overlay")"
mkdir -p "$(dirname "$overlay")"
mkdir "$overlay"
stock_btusb=$(modinfo -b "$image" -k "$abi" -F filename btusb)
stock_btmtk=$(modinfo -b "$image" -k "$abi" -F filename btmtk)
for module in "${modules[@]}"; do cp "candidate/$module.ko" "$overlay/"; done
index_tree candidate
lookup "$@" > candidate/lookup-candidate.log
for module in "${modules[@]}"; do
    test "$(modinfo -b "$image" -k "$abi" -F filename "$module")" = "$overlay/$module.ko"
    grep -Fq "insmod $overlay/$module.ko" candidate/lookup-candidate.log
done
awk -v usb="${stock_btusb#"$tree/"}" -v mtk="${stock_btmtk#"$tree/"}" '
    { gsub(/updates\/vrooli-mt6639\/btusb\.ko/, usb); gsub(/updates\/vrooli-mt6639\/btmtk\.ko/, mtk); print }
' candidate/modules-candidate.dep | sort > candidate/modules-normalized.dep
cmp <(sort candidate/modules-stock.dep) candidate/modules-normalized.dep
du -B1 -s "$work" | cut -f1 > candidate/allocated-with-overlay.bytes

for module in "${modules[@]}"; do unlink "$overlay/$module.ko"; done
rmdir "$overlay"
index_tree rollback
lookup "$@" > candidate/lookup-rollback.log
hash_tree rollback
cmp candidate/tree-stock.sha256 candidate/tree-rollback.sha256
cmp candidate/tree-stock.links candidate/tree-rollback.links
cmp candidate/lookup-stock.log candidate/lookup-rollback.log
