// Hand-written so Swift sees only the Xray entry points; the WireGuard
// functions in the same archive are declared by WireGuardKit's own
// wireguard.h (module WireGuardKitGo).
#ifndef HEXVPN_LIBXRAY_BRIDGE_H
#define HEXVPN_LIBXRAY_BRIDGE_H

#ifdef __cplusplus
extern "C" {
#endif

/// libXray Invoke: takes the JSON envelope {apiVersion:3, method, payload},
/// returns a malloc'd JSON response. Free it with CGoFree.
char *CGoInvoke(const char *requestJSON);
void CGoFree(char *value);

#ifdef __cplusplus
}
#endif

#endif
