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
    socket = "~/.nixos-service.sock";

    curlCommand = pkgs.writeScript "upload-to-cache.sh" ''
      #!/bin/sh
      set -eu
      set -f # disable globbing
      export IFS=' '

      echo "Uploading paths $OUT_PATHS"

      ${lib.getExe pkgs.nixos-service} upload -s "${socket}" "$OUT_PATHS" || true
    '';
  in
    mkIf cfg.enable {
      nix.settings = {
        post-build-hook = curlCommand;
        trusted-users = ["${cfg.user}"];
      };

      environment.variables.NIXOS_SERVICE_SOCK_PATH = socket;

      # users = {
      #   users = {
      #     "${cfg.user}" = {
      #       group = cfg.group;
      #       isSystemUser = true;
      #     };
      #     root.extraGroups = [cfg.group];
      #   };
      #   groups."${cfg.group}" = {};
      # };

      systemd.user = {
        sockets.nixos-service = {
          description = "Socket to communicate with myservice";
          listenStreams = [socket];

          # socketConfig = {
          #   SocketUser = cfg.user;
          #   SocketGroup = cfg.group;
          #   SocketMode = cfg.mode;
          #   DirectoryMode = cfg.mode;
          # };
        };

        services.nixos-service = {
          description = "nixos-service";
          enable = true;

          wantedBy = ["multi-user.target"];
          # requires = ["nixos-service.socket"];

          path = with pkgs; [attic-client dbus];

          environment = {
            NIXOS_SERVICE_ATTIC_SERVER_NAME = cfg.serverName;
            NIXOS_SERVICE_ATTIC_URL = cfg.serverUrl;
            NIXOS_SERVICE_ATTIC_SECRET_PATH = cfg.secretPath;
            NIXOS_SERVICE_SOCK_PATH = socket;
            # XDG_CONFIG_HOME = "/var/lib/nixos-service";
          };

          serviceConfig = {
            # User = cfg.user;
            # Group = cfg.group;
            # RuntimeDirectory = runtimeDirectory;
            # RuntimeDirectoryMode = cfg.mode;
            # StateDirectory = "nixos-service";
            # StateDirectoryMode = cfg.mode;
            ExecStart = "${lib.getExe pkgs.nixos-service} socket";
            Restart = "on-failure";
            RestartSec = 5;
          };
        };
      };
    };
}
