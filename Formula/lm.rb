class Lm < Formula
  desc "Focused Lunch Money v2 CLI for transaction review workflows"
  homepage "https://github.com/muinmomin/lunchmoney-cli"
  version "0.1.4"

  on_macos do
    on_arm do
      url "https://github.com/muinmomin/lunchmoney-cli/releases/download/v#{version}/lm-darwin-arm64.tar.gz"
      sha256 "9975925c294e123c307d926dd9db2975cc4fb3d7d9a8d2690617ec787dac0ab4"
    end

    on_intel do
      url "https://github.com/muinmomin/lunchmoney-cli/releases/download/v#{version}/lm-darwin-amd64.tar.gz"
      sha256 "d79591c5fcac91716aa686f043314adc8ef31a89547ee96f140e8839261b7b3a"
    end
  end

  def install
    bin.install "lm"
  end

  test do
    assert_match "Lunch Money CLI", shell_output("#{bin}/lm --help")
  end
end
