# Tendermint Director

## Overview
Director is a server application that helps coordination with Tendermint-based blockchain network setup.
It is part of the open-source toolset provided by [Testnet Kitchen](https://testnet.kitchen).

Coordination among testnet peers can become a cumbersome process: get everyone's public key and distribute the genesis file.
Director is a service that accepts public key registrations for future testnets and when enough validators are registered
it provides the compiled genesis. The testnet participants can set up their own nodes at their leisure.

## How to get it
You can download the latest release from the [releases page](https://github.com/testnetkitchen/director/releases).

## How to build it
You can build it by running:
```bash
git clone https://github.com/testnetkitchen/director
cd director
go -o director cmd/director/main.go
```

## How to run
```bash
./director init
./director node
```

The `init` command will initialize a default config file under `$HOME/.director/config/config.toml`.

The `node` command will start the web service and listen for incoming requests (by default on port 27001). Use the examples
in the [client](https://github.com/testnetkitchen/director/tree/master/client) folder to interact with the server.

The home directory can be changed with the `--home` flag.

## How does it work
The `config.toml` is self-explaining.

The default configuration defines a network with the chain ID `default`. (You can add more testnet configurations in the config.)

The default service expects 4 validators to be registered. After 4 sucessful registrations it starts compiling the genesis file.
At that point no more registrations are accepted.

If less than 4 validators register within the default 2 hours, it closes registration and compiles the genesis with the existing registrations.

Both the number of expected validators and the registration period (timeout) can be configured in the config file.

