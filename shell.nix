with import <nixpkgs> {};

stdenv.mkDerivation {
name = "go-env";

buildInputs = [
    go
    syft
    grype
    docker
    trivy
];

SOURCE_DATE_EPOCH = 315532800;
PROJDIR = "${toString ./.}";
S_NETWORK="host";

shellHook = ''
    export LD_LIBRARY_PATH="${pkgs.stdenv.cc.cc.lib}/lib"
    export PATH=/tmp/bin:$PATH
    mkdir /tmp/bin
    '';
}
