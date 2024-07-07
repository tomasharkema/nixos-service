{buildGoModule}:
buildGoModule rec {
  pname = "nixos-service";
  version = "0.0.1";

  src = ./src;

  vendorHash = "sha256-E5XvYJiMz3yBcCTAMEJeowOFCpDZzS5RzSpbMe7qHlk=";

  meta.mainProgram = "${pname}";
}
