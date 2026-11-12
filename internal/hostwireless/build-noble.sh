#!/usr/bin/env bash
# Exact Noble backport recipe. The control-plane owner supplies only private
# working paths and observed PCI aliases, under a bounded disk reservation.
set -Eeuo pipefail
trap 'printf "wireless recipe stopped at line %s\n" "$LINENO" >&2' ERR
umask 077
phase="$1"
abi="$2"
shift 2
work=$PWD
headers="/lib/modules/$abi/build"
source="$work/ubuntu-source"
mt76="$source/drivers/net/wireless/mediatek/mt76"
modules=(mt76-connac-lib mt792x-lib mt7925-common mt7925e)

test "$(id -u)" -ne 0
test "$(uname -r)" = "$abi"
test "$abi" = "7.0.0-30-generic"

build() {
    local out="$1" module
    mkdir "$out"
    make -j4 -C "$headers" "M=$mt76" CC=x86_64-linux-gnu-gcc-13 \
        CONFIG_MT76_CORE=n CONFIG_MT76_USB=n CONFIG_MT76_SDIO=n \
        CONFIG_MT76x02_LIB=n CONFIG_MT76x02_USB=n CONFIG_MT792x_USB=n \
        CONFIG_MT76x0_COMMON=n CONFIG_MT76x2_COMMON=n CONFIG_MT7603E=n \
        CONFIG_MT7615_COMMON=n CONFIG_MT7915E=n CONFIG_MT7921_COMMON=n \
        CONFIG_MT7996E=n CONFIG_MT7925U=n modules > "$out/build.log" 2>&1 ||
        { tail -60 "$out/build.log" >&2; return 1; }
    cp "$mt76/Module.symvers" "$mt76/modules.order" "$out/"
    test "$(wc -l < "$out/modules.order")" -eq "${#modules[@]}"
    for module in "${modules[@]}"; do
        local paths=()
        mapfile -t paths < <(find "$mt76" -type f -name "$module.ko")
        test "${#paths[@]}" -eq 1
        grep -Eq "(^|/)$module\.o$" "$out/modules.order"
        cp "${paths[0]}" "$out/"
        modprobe --show-modversions "${paths[0]}" > "$out/$module.modversions"
    done
}

if [[ "$phase" == stock ]]; then
    cmp "$headers/.config" "/boot/config-$abi"
    sha256sum "$headers/Module.symvers" "$headers/.config" "/boot/config-$abi" > kernel-inputs.sha256
    subtree=drivers/net/wireless/mediatek/mt76
    mkdir "$source"
    {
        tar -xzf inputs/linux-hwe-7.0_7.0.0.orig.tar.gz --strip-components=1 -C "$source" "linux-7.0/$subtree"
        gzip -cd inputs/linux-hwe-7.0_7.0.0-30.30~24.04.1.diff.gz |
            git -C "$source" apply -p1 --include="$subtree/**"
        (cd "$source" && find "$subtree" -type f -print0 | sort -z | xargs -0 sha256sum) > stock-source.sha256
        # Manifest independently reproduced from the full dpkg-source baseline.
        sha256sum --check <<< "7db572d4d5a7078c6d03a090781e80b2e8f7b5a8e248b5e40b16c6fcb559e660  stock-source.sha256"
    } > unpack.log
    build stock
    exit
fi

test "$phase" = candidate
test "$#" -gt 0
sha256sum --check kernel-inputs.sha256
git -C "$source" apply --check "$work/adapted.patch"
git -C "$source" apply "$work/adapted.patch"
build candidate
sha256sum --check kernel-inputs.sha256
cp kernel-inputs.sha256 candidate/
cp unpack.log candidate/
cp stock-source.sha256 candidate/
cp stock/Module.symvers candidate/stock.Module.symvers
cp stock/build.log candidate/stock-build.log

image="$work/copied-tree"
tree="$image/lib/modules/$abi"
overlay="$tree/updates/vrooli-mt7927"
mkdir -p "$image/lib/modules"
cp -a --reflink=auto "/lib/modules/$abi" "$image/lib/modules/"
test -d "$tree" && test ! -L "$tree"
unlink "$tree/build"
while IFS= read -r -d '' link; do
    target=$(readlink -f "$link")
    [[ "$target" == "$tree/"* ]]
done < <(find "$tree" -type l -print0)
cp /etc/depmod.d/ubuntu.conf "$work/depmod.conf"
lookup() {
    local module alias
    for module in "${modules[@]}"; do
        printf '\nmodule %s\n' "$module"
        modinfo -b "$image" -k "$abi" -F filename "$module"
        modprobe -C /dev/null -d "$image" -S "$abi" --show-depends "$module"
    done
    for alias; do
        printf '\nmodalias %s\n' "$alias"
        modprobe -C /dev/null -d "$image" -S "$abi" --resolve-alias "$alias" || true
        modprobe -C /dev/null -d "$image" -S "$abi" --show-depends "$alias" || true
    done
}
index_tree() {
    local stage="$1"
    depmod -b "$image" -C "$work/depmod.conf" -a -e -E "$headers/Module.symvers" "$abi" > "candidate/depmod-$stage.log" 2>&1
    if test -s "candidate/depmod-$stage.log"; then
        tail -60 "candidate/depmod-$stage.log" >&2
        return 1
    fi
}
hash_tree() {
    (cd "$tree" && find . -type f -print0 | sort -z | xargs -0 -r sha256sum) > "candidate/tree-$1.sha256"
    (cd "$tree" && find . -type l -printf '%p -> %l\n' | sort) > "candidate/tree-$1.links"
}
other_providers() {
    # modules.dep lists the selected path for every indexed provider.
    cut -d: -f1 "$tree/modules.dep" |
        awk '!/(^|\/)(mt76-connac-lib|mt792x-lib|mt7925-common|mt7925e)\.ko(\.(zst|xz|gz))?$/' |
        sort > "candidate/providers-$1.txt"
}

index_tree stock
lookup "$@" > candidate/lookup-stock.log 2>&1
hash_tree stock
other_providers stock
test ! -L "$(dirname "$overlay")"
mkdir -p "$(dirname "$overlay")"
mkdir "$overlay"
for module in "${modules[@]}"; do
    f="$work/candidate/$module.ko"
    test "$(modinfo -F vermagic "$f")" = "$(modinfo -k "$abi" -F vermagic "$module")"
    test -z "$(modinfo -F signer "$f")"
    modinfo "$f" >> candidate/modinfo.txt
    cp "$f" "$overlay/"
done
index_tree candidate
lookup "$@" > candidate/lookup-candidate.log 2>&1
du -B1 -s "$work" | cut -f1 > candidate/allocated-with-overlay.bytes
for alias; do
    test "$(modprobe -C /dev/null -d "$image" -S "$abi" --resolve-alias "$alias")" = mt7925e
done
firmware=$(modinfo -F firmware candidate/mt7925e.ko)
grep -Fxq 'mediatek/mt7927/WIFI_MT6639_PATCH_MCU_2_1_hdr.bin' <<< "$firmware"
grep -Fxq 'mediatek/mt7927/WIFI_RAM_CODE_MT6639_2_1.bin' <<< "$firmware"
for module in "${modules[@]}"; do
    test "$(modinfo -b "$image" -k "$abi" -F filename "$module")" = "$overlay/$module.ko"
done
other_providers candidate
cmp candidate/providers-stock.txt candidate/providers-candidate.txt
for module in "${modules[@]}"; do unlink "$overlay/$module.ko"; done
rmdir "$overlay"
index_tree rollback
lookup "$@" > candidate/lookup-rollback.log 2>&1
hash_tree rollback
cmp candidate/tree-stock.sha256 candidate/tree-rollback.sha256
cmp candidate/tree-stock.links candidate/tree-rollback.links
cmp candidate/lookup-stock.log candidate/lookup-rollback.log
