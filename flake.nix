{
  description = "A language aquisition terminal application (TUI) to learn languages with text";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    systems.url = "github:nix-systems/default";

    treefmt-nix = {
      url = "github:numtide/treefmt-nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };

    gomod2nix = {
      url = "github:nix-community/gomod2nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    {
      self,
      nixpkgs,
      systems,
      treefmt-nix,
      gomod2nix,
      ...
    }:
    let
      forAllSystems = nixpkgs.lib.genAttrs (import systems);

      treefmtEval = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        treefmt-nix.lib.evalModule pkgs ./nix/treefmt.nix
      );
    in
    {
      formatter = forAllSystems (system: treefmtEval.${system}.config.build.wrapper);

      checks = forAllSystems (system: {
        formatting = treefmtEval.${system}.config.build.check self;
      });

      devShells = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
          goEnv = gomod2nix.legacyPackages.${system}.mkGoEnv { pwd = ./.; };
        in
        import ./nix/shell.nix {
          inherit pkgs goEnv;
          inherit (gomod2nix.legacyPackages.${system}) gomod2nix;
        }
      );

      packages = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          default = self.packages.${system}.lingo;
          lingo = pkgs.callPackage ./nix/pkgs/lingo {
            inherit (gomod2nix.legacyPackages.${system}) buildGoApplication;
          };
          inherit (gomod2nix.legacyPackages.${system}) gomod2nix;
        }
      );
    };
}
