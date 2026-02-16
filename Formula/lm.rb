class Lm < Formula
  desc "Focused Lunch Money v2 CLI for transaction review workflows"
  homepage "https://github.com/muinmomin/lunchmoney-cli"
  version "0.1.3"

  on_macos do
    on_arm do
      url "https://github.com/muinmomin/lunchmoney-cli/releases/download/v#{version}/lm-darwin-arm64.tar.gz"
      sha256 "fbae7ad7e2e0a489c848d10b0997b38d8b8dea833a529644b8fee2380589487a"
    end

    on_intel do
      url "https://github.com/muinmomin/lunchmoney-cli/releases/download/v#{version}/lm-darwin-amd64.tar.gz"
      sha256 "c0c92bfff4299166435e0c68e8496e59e205bbbc01f19e90d2795dae616a6581"
    end
  end

  def install
    bin.install "lm"
  end

  test do
    assert_match "Lunch Money CLI", shell_output("#{bin}/lm --help")
  end
end
