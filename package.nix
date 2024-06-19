{buildGoModule}:
buildGoModule rec {
  pname = "nixos-service";
  version = "0.0.1";

  src = ./src;

  vendorHash = "sha256-ToAVii2yp+69PUwdpJr98S554yizQr+o0jB6XRKpI84=";

  meta.mainProgram = "${pname}";
}
