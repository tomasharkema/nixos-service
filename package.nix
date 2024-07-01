{buildGoModule}:
buildGoModule rec {
  pname = "nixos-service";
  version = "0.0.1";

  src = ./src;

  vendorHash = "sha256-yknwiZBXGLxaT5iHIsahx+dkd0viGV9Mo2ku99bUicM=";

  meta.mainProgram = "${pname}";
}
