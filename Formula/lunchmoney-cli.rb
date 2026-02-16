class LunchmoneyCli < Formula
  desc "Focused Lunch Money v2 CLI for transaction review workflows"
  homepage "https://github.com/muinmomin/lunchmoney-cli"
  version "0.0.0"

  on_macos do
    on_arm do
      url "https://github.com/muinmomin/lunchmoney-cli/releases/download/v#{version}/lm-darwin-arm64.tar.gz"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    end

    on_intel do
      url "https://github.com/muinmomin/lunchmoney-cli/releases/download/v#{version}/lm-darwin-amd64.tar.gz"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    end
  end

  def install
    bin.install "lm"
  end

  test do
    assert_match "Lunch Money CLI", shell_output("#{bin}/lm --help")
  end
end
