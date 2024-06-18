{self, ...}: final: prev: {
  nixos-service = self.packages."${prev.system}".nixos-service;
}
