{
  description = "A Simple Plan — minimalist static site generator";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in {
        packages.default = pkgs.buildGoModule {
          pname = "plan";
          version = "0.1.0";
          src = ./.;
          subPackages = [ "cmd/plan" ];
          vendorHash = "sha256-25ZAjn5dwkPQfG90yXsemFy1pf0n85yFqwcKQUvrDOo=";
        };

        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go
            gopls
            gotools
            git
          ];
        };
      });
}
