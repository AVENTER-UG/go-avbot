with import <nixpkgs> {};

stdenv.mkDerivation {
name = "go-env";

buildInputs = [
    syft
    grype
    docker
    trivy
		stdenv.cc.cc
		libpcap
		docker-credential-helpers
		dbus
		sudo
];

SOURCE_DATE_EPOCH = 315532800;
PROJDIR = "${toString ./.}";
S_NETWORK="host";
S_IMAGE="localhost:5000/debian_build:12";

shellHook = ''
    export LD_LIBRARY_PATH="${pkgs.stdenv.cc.cc.lib}/lib"
    export PATH=/tmp/bin:$PATH
		export GOTMPDIR=/tmp
		export TMPDIR=/tmp
    mkdir /tmp/bin
    '';
}
