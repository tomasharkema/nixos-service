{
  description = "A very basic flake";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs?ref=nixos-unstable";

    systems.url = "github:nix-systems/default";
  };

  outputs = {
    self,
    nixpkgs,
    systems,
    ...
  }: let
    eachSystem = nixpkgs.lib.genAttrs (import systems);
  in {
    nixosModules.nixos-service = import ./module.nix;
    nixosModules.default = self.nixosModules.nixos-service;

    packages = eachSystem (system: {
      nixos-service = nixpkgs.legacyPackages.${system}.callPackage ./package.nix {};
      default = self.packages."${system}".nixos-service;
    });

    overlays.default = import ./overlay.nix;

    inherit nixpkgs;
  };
}
