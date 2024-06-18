{self, ...}: final: prev: rec {
  nixos-service = self.packages."${prev.system}".default;
}
