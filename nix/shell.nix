{
  pkgs,
  gomod2nix,
  goEnv,
}:

{
  default = pkgs.mkShell {
    nativeBuildInputs = [
      pkgs.go
      pkgs.gopls
      pkgs.gotools
      pkgs.go-tools
      pkgs.goimports-reviser
      gomod2nix
      goEnv

      pkgs.pkg-config
      pkgs.alsa-lib

      (pkgs.writeShellScriptBin "update-modules" ''
        go get -u
        go mod tidy
        nix run .#gomod2nix -- generate --outdir nix/pkgs/lingo
      '')
    ];

    shellHook = ''
      echo -e "\033[0;32;4mHeper commands:\033[0m"
      echo "'update-modules' instead of 'go get -u && go mod tidy'"
    '';
  };
}
