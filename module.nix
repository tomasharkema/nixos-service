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

    mode = mkOption {
      type = types.str;
      default = "0755";
    };
  };

  config = let
    runtimeDirectory = "nixos-service";
    socket = "/run/${runtimeDirectory}/nixos-service.sock";

    curlCommand = pkgs.writeScript "upload-to-cache.sh" ''
      #!/bin/sh
      set -x
      set -eu
      set -f # disable globbing
      export IFS=' '

      echo "Uploading paths $OUT_PATHS"

      exec ${lib.getExe pkgs.nixos-service} upload -s "${socket}" "$OUT_PATHS"
    '';
    # $NIXOS_SERVICE_SOCK_PATH
  in
    mkIf cfg.enable {
      nix.settings.post-build-hook = curlCommand;

      environment.variables.NIXOS_SERVICE_SOCK_PATH = socket;

      users.groups."${cfg.group}" = {};

      systemd.sockets.nixos-service = {
        description = "Socket to communicate with myservice";
        listenStreams = [socket];

        socketConfig = {
          SocketGroup = cfg.group;
          SocketMode = cfg.mode;
          DirectoryMode = cfg.mode;
        };
      };

      systemd.services.nixos-service = {
        description = "nixos-service";
        enable = true;

        wantedBy = ["multi-user.target"];

        path = [pkgs.attic-client pkgs.dbus];

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
          RuntimeDirectoryMode = cfg.mode;
          ExecStart = "${lib.getExe pkgs.nixos-service} socket";
          Restart = "on-failure";
          RestartSec = 5;
        };
      };
    };
}
