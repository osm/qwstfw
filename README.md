# qwstfw

QuakeWorld stufftext firewall.

## Why

Quake servers are, by design, allowed to push commands to clients that will be
executed. This was not an issue in the original client; however, due to
additional features being added to the client that do not take the possibility
of remote execution into consideration, it has become vulnerable to various
security issues.

This video demonstrates the, still unpatched, security issue in ezQuake.
https://www.youtube.com/watch?v=13Oh1960MFs

## How

`qwstfw` intercepts all `stufftext` commands sent by the server, validates
them against a set of allowed commands. If it notices a command that isn't
explicitly allowed, it will block the given command.

## Usage with ezQuake

1. Start `qwstfw`
2. Launch `ezQuake` and set `cl_proxyaddr` to `127.0.0.1:27500`
3. Type `/connect <server>`

Note: you can chain multiple proxies by setting `cl_proxyaddr` to
`127.0.0.1:27500@address.to.second.proxy.com:27500`.

## Downloads

When a client connects to a server that uses a map (or any other asset) your
client doesn't have, it will try to download the asset from the server. This
can pose a security risk, as a malicious server could upload an executable
file and trigger its execution with subsequent commands.

Therefore, client-side downloads are disabled by default when using `qwstfw`.
However, this can be inconvenient, so you can toggle client-side downloads by
using the `-allow-downloads` command-line flag or setting the
`allow_downloads` option in the config file.

I recommend that you keep this disabled and instead download your assets from
a trusted source, and install them manually in your Quake directory.

## Commands

`qwstfw` reads the set of allowed commands from its config file. You can
review and edit these commands as needed; they are located under the
`[commands]` section in the config file.

## Aliases

To make life easier for KTX players, we inject a set of default aliases. This
is necessary because `qwstfw` doesn't allow servers to push `alias` to your
client, as this can easily be abused for malicious purposes.

While the default aliases we inject should suffice, they may need to be
updated if new commands are added to KTX. Therefore, you can provide
additional aliases or commands to inject when your connect to a server.

Add your additional aliases under the `[aliases]` section in the config file.

`qwstfw` injects the defined aliases to the client when it receives a
`on_enter` or `on_spec_enter` stufftext from the server.

## QTV

Unfortunately, there is no QTV support yet, so you need to be aware that this
proxy does not secure your client from malicious QTV servers.

## Advanced

If you are interested in knowing which commands are allowed and/or blocked,
you can start `qwstfw` with the `-verbose` flag. When doing so, it will output
this information in the console.
