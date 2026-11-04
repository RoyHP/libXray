import os
import subprocess

from app.build import Builder


class AppleGoMobileBuilder(Builder):
    def before_build(self):
        # HexVPN: no download_geo — the clients carry no geo assets and the
        # framework never embedded them anyway (see distro/distro.go).
        self.prepare_xray_core()
        self.init_go_env()
        self.clean_lib_dirs(["LibXray.xcframework"])
        self.prepare_gomobile()

    def build(self):
        self.snapshot_go_env()
        try:
            self.before_build()

            os.chdir(self.lib_dir)
            ret = subprocess.run(
                [
                    "gomobile",
                    "bind",
                    "-target",
                    # HexVPN: only the slices the iOS app links (no x86_64
                    # simulator, no macOS/Catalyst).
                    "ios/arm64,iossimulator/arm64",
                    # Strip symbol tables + DWARF from the Go archive; the
                    # extension must stay small (jetsam counts mapped text
                    # too) and nothing debugs Go frames from Xcode.
                    "-trimpath",
                    "-ldflags=-s -w -buildid=",
                    "-iosversion",
                    "15.0",
                ]
            )
            if ret.returncode != 0:
                raise Exception("build failed")
        finally:
            self.restore_go_env()
