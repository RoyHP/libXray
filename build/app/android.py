# app/android.py
import os
import shutil
import subprocess

from app.build import Builder

# HexVPN: the Android AAR carries BOTH engines of the mobile app: libXray
# (HexGamma, VLESS+REALITY) and hexwg (HexGuard, wireguard-go over the
# VpnService's tun fd), bound together so the app process has ONE Go
# runtime. wireguard-go's keepalive/handshake timers must keep counting
# through deep sleep, which stock Go's monotonic clock does not on Android
# (it stops in suspend); wireguard-android builds its libwg-go.so against a
# Go runtime patched to use CLOCK_BOOTTIME, and so does our iOS bridge
# (hexvpn_bridge.sh). The same vendored diff is applied here to a copy of
# GOROOT before gomobile runs.
BOOTTIME_PATCH = os.path.join("hexvpn_bridge", "goruntime-boottime-over-monotonic.diff")
PACKAGES = [".", "./hexwg"]
# arm64 phones + the Apple-silicon emulator; armeabi-v7a for the long tail.
# x86_64 is deliberately absent (no x86 emulator here; ~20 MB per ABI).
TARGET = "android/arm64,android/arm"


class AndroidBuilder(Builder):

    def before_build(self):
        super().before_build()
        self.prepare_gomobile()

    def prepare_patched_goroot(self) -> str:
        real = subprocess.run(
            ["go", "env", "GOROOT"], capture_output=True, text=True, check=True
        ).stdout.strip()
        stamp = subprocess.run(
            ["go", "version"], capture_output=True, text=True, check=True
        ).stdout.strip()
        goroot = os.path.join(self.lib_dir, "hexvpn_bridge", ".goroot-android")
        stamp_file = os.path.join(goroot, ".prepared")
        if os.path.isfile(stamp_file):
            with open(stamp_file) as f:
                if f.read().strip() == stamp:
                    return goroot
        print(f"==> preparing patched GOROOT for Android ({stamp})")
        shutil.rmtree(goroot, ignore_errors=True)
        os.makedirs(goroot)
        subprocess.run(
            ["rsync", "-a", "--delete", "--exclude=pkg/obj/go-build", real + "/", goroot + "/"],
            check=True,
        )
        subprocess.run(
            ["patch", "-p1", "-d", goroot],
            stdin=open(os.path.join(self.lib_dir, BOOTTIME_PATCH)),
            check=True,
        )
        with open(stamp_file, "w") as f:
            f.write(stamp)
        return goroot

    def build(self):
        self.snapshot_go_env()
        try:
            self.before_build()

            clean_files = ["libXray-sources.jar", "libXray.aar"]
            self.clean_lib_files(clean_files)
            os.chdir(self.lib_dir)
            env = os.environ.copy()
            env["GOROOT"] = self.prepare_patched_goroot()
            ret = subprocess.run(
                [
                    "gomobile",
                    "bind",
                    "-target",
                    TARGET,
                    "-androidapi",
                    "24",
                    "-trimpath",
                    "-ldflags=-checklinkname=0 -s -w -buildid= "
                    "-extldflags=-Wl,-z,max-page-size=16384",
                    "-o",
                    "libXray.aar",
                ]
                + PACKAGES,
                env=env,
            )
            if ret.returncode != 0:
                raise Exception("build failed")
        finally:
            self.restore_go_env()
