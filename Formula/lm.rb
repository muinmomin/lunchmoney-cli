class Lm < Formula
  desc "Focused Lunch Money v2 CLI for transaction review workflows"
  homepage "https://github.com/muinmomin/lunchmoney-cli"
  version "0.1.5"

  on_macos do
    on_arm do
      url "https://github.com/muinmomin/lunchmoney-cli/releases/download/v#{version}/lm-darwin-arm64.tar.gz"
      sha256 "1fb40b76dc788ad1913d6badfc9dfb9322d66fcc11a9f6be9412e699b7eacc80"
    end

    on_intel do
      url "https://github.com/muinmomin/lunchmoney-cli/releases/download/v#{version}/lm-darwin-amd64.tar.gz"
      sha256 "174aa0c8021ddc715e811e65167f1e72f8a40017f876fe766d1af63b8e3c40f8"
    end
  end

  def install
    bin.install "lm"
  end

  test do
    assert_match "Lunch Money CLI", shell_output("#{bin}/lm --help")
  end
end
