{
  config,
  options,
  lib,
  pkgs,
  ...
}:
with lib; let
  cfg = config.services.nixos-service;
in {
  options.services.nixos-service = {
    enable = mkOption {
      type = types.bool;
      default = false;
    };

    user = mkOption {
      type = types.str;
      default = "nixos-service";
    };

    group = mkOption {
      type = types.str;
      default = "nixos-service";
    };

    serverName = mkOption {
      type = types.str;
      # default=  "tomas";
    };

    serverUrl = mkOption {
      type = types.str;
      # default="tomas";
    };

    secretPath = mkOption {
      type = types.str;
    };
  };

  config = let
    runtimeDirectory = "nixos-service";
    socket = "/run/${runtimeDirectory}/nixos-service.sock";

    # atPostBuildScript = pkgs.writeScript "upload-to-cache-at.sh" ''
    #   {
    #     echo "Uploading paths $@"
    #     ${pkgs.attic}/bin/attic login "${cfg.serverName}" "${cfg.serverAddress}" "$(cat "${config.age.secrets.attic-key.path}")"
    #     ${pkgs.attic}/bin/attic push "${cfg.serverName}:tomas" -j1 $@
    #     echo "Uploaded paths $@"
    #   } |& tee /var/log/attic-upload.log
    # '';
    # postBuildScript = pkgs.writeScript "upload-to-cache.sh" ''
    #   set -eu
    #   set -f # disable globbing
    #   export IFS=' '

    #   echo "Uploading paths $OUT_PATHS"

    #   echo "${pkgs.su}/bin/su -c '${atPostBuildScript} $OUT_PATHS' tomas" | ${pkgs.at}/bin/at -m -q b now
    # '';
    curlCommand = pkgs.writeScript "upload-to-cache.sh" ''
      set -eu
      set -f # disable globbing
      export IFS=' '

      echo "Uploading paths $OUT_PATHS"

      exec ${pkgs.curl}/bin/curl --unix-socket $NIXOS_SERVICE_SOCK_PATH http://localhost -d "$OUT_PATHS" 2>/dev/null
    '';
  in
    mkIf cfg.enable {
      nix.settings.post-build-hook = curlCommand;

      environment.variables.NIXOS_SERVICE_SOCK_PATH = socket;

      users.groups."${cfg.group}" = {};

      systemd.services.nixos-service = {
        description = "nixos-service";

        wantedBy = ["multi-user.target"];

        environment = {
          NIXOS_SERVICE_ATTIC_NAME = "nixos-service";
          NIXOS_SERVICE_ATTIC_SERVER_NAME = "nixos-service:${cfg.serverName}";
          NIXOS_SERVICE_ATTIC_URL = cfg.serverUrl;
          NIXOS_SERVICE_ATTIC_SECRET_PATH = cfg.secretPath;
          NIXOS_SERVICE_SOCK_PATH = socket;
        };

        serviceConfig = {
          Group = cfg.group;
          RuntimeDirectory = runtimeDirectory;
          RuntimeDirectoryMode = "0755";
          ExecStart = "${lib.getExe pkgs.nixos-service}";
        };

        socketConfig = {
          ListenStream = cfg.socket;
        };
      };
    };
}
