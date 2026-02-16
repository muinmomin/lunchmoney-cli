class Lm < Formula
  desc "Focused Lunch Money v2 CLI for transaction review workflows"
  homepage "https://github.com/muinmomin/lunchmoney-cli"
  version "0.1.1"

  on_macos do
    on_arm do
      url "https://github.com/muinmomin/lunchmoney-cli/releases/download/v#{version}/lm-darwin-arm64.tar.gz"
      sha256 "a96b83cca913a977aa12158545af0a3a410cd74d57113fcab76f7aaa47836cd6"
    end

    on_intel do
      url "https://github.com/muinmomin/lunchmoney-cli/releases/download/v#{version}/lm-darwin-amd64.tar.gz"
      sha256 "f1fe52f6f206b4ae9832b41b57539875d4f85d0d83ce3f696015be4a76502bcf"
    end
  end

  def install
    bin.install "lm"
  end

  test do
    assert_match "Lunch Money CLI", shell_output("#{bin}/lm --help")
  end
end
