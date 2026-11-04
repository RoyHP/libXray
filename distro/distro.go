// Package distro registers the subset of Xray-core features HexVPN's mobile
// clients use. It replaces github.com/xtls/xray-core/main/distro/all, which
// links every inbound/outbound/transport plus the commander, observatory,
// stats, fakedns and geodata machinery — none of which runs inside a 50 MiB
// iOS packet tunnel. The config loader is JSON only; there are no geoip /
// geosite loaders and no geo assets anywhere in the client.
package distro

import (
	// Mandatory.
	_ "github.com/xtls/xray-core/app/dispatcher"
	_ "github.com/xtls/xray-core/app/proxyman/inbound"
	_ "github.com/xtls/xray-core/app/proxyman/outbound"

	// Config sections we emit: log, policy, routing.
	_ "github.com/xtls/xray-core/app/log"
	_ "github.com/xtls/xray-core/app/policy"
	_ "github.com/xtls/xray-core/app/router"

	// Breaks the internet <-> core import cycle for tagged dialers.
	_ "github.com/xtls/xray-core/transport/internet/tagged/taggedimpl"

	// Inbound: the TUN fed by the NetworkExtension / VpnService fd.
	_ "github.com/xtls/xray-core/proxy/tun"

	// Outbounds: HexGamma (vless+reality+vision), HexGuard (wireguard),
	// blackhole for the IPv6 / private-range rules, freedom for probes.
	_ "github.com/xtls/xray-core/proxy/blackhole"
	_ "github.com/xtls/xray-core/proxy/freedom"
	_ "github.com/xtls/xray-core/proxy/vless/outbound"
	_ "github.com/xtls/xray-core/proxy/wireguard"

	// Transports.
	_ "github.com/xtls/xray-core/transport/internet/reality"
	_ "github.com/xtls/xray-core/transport/internet/tcp"
	_ "github.com/xtls/xray-core/transport/internet/tls"
	_ "github.com/xtls/xray-core/transport/internet/udp"

	// JSON config loader.
	_ "github.com/xtls/xray-core/main/json"
)
