final: prev: {
  nixos-service = prev.pkgs.callPackage ./package.nix {};
}
