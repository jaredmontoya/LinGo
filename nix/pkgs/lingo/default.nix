{
  lib,
  pkg-config,
  alsa-lib,
  buildGoApplication,
}:

buildGoApplication {
  pname = "lingo";
  version = "0.0.1";
  src = ../../../.;
  pwd = ../../../.;
  modules = ./gomod2nix.toml;

  nativeBuildInputs = [ pkg-config ];

  buildInputs = [ alsa-lib ];

  ldflags = [
    "-s"
    "-w"
  ];

  meta = with lib; {
    description = "A language aquisition terminal application (TUI) to learn languages with text";
    homepage = "https://github.com/danielmiessler/fabric";
    license = licenses.mit;
    platforms = platforms.all;
    mainProgram = "lingo";
  };
}
