package capabiliities

import (
	"kernel.org/pub/linux/libs/security/libcap/cap"
)

var Capabilities = map[string]cap.Value{
	"CAP_AUDIT_CONTROL":      cap.AUDIT_CONTROL,
	"CAP_AUDIT_READ":         cap.AUDIT_READ,
	"CAP_AUDIT_WRITE":        cap.AUDIT_WRITE,
	"CAP_BLOCK_SUSPEND":      cap.BLOCK_SUSPEND,
	"CAP_BPF":                cap.BPF,
	"CAP_CHECKPOINT_RESTORE": cap.CHECKPOINT_RESTORE,
	"CAP_CHOWN":              cap.CHOWN,
	"CAP_DAC_OVERRIDE":       cap.DAC_OVERRIDE,
	"CAP_DAC_READ_SEARCH":    cap.DAC_READ_SEARCH,
	"CAP_FOWNER":             cap.FOWNER,
	"CAP_FSETID":             cap.FSETID,
	"CAP_IPC_LOCK":           cap.IPC_LOCK,
	"CAP_IPC_OWNER":          cap.IPC_OWNER,
	"CAP_KILL":               cap.KILL,
	"CAP_LEASE":              cap.LEASE,
	"CAP_LINUX_IMMUTABLE":    cap.LINUX_IMMUTABLE,
	"CAP_MAC_ADMIN":          cap.MAC_ADMIN,
	"CAP_MAC_OVERRIDE":       cap.MAC_OVERRIDE,
	"CAP_MKNOD":              cap.MKNOD,
	"CAP_NET_ADMIN":          cap.NET_ADMIN,
	"CAP_NET_BIND_SERVICE":   cap.NET_BIND_SERVICE,
	"CAP_NET_BROADCAST":      cap.NET_BROADCAST,
	"CAP_NET_RAW":            cap.NET_RAW,
	"CAP_PERFMON":            cap.PERFMON,
	"CAP_SETGID":             cap.SETGID,
	"CAP_SETFCAP":            cap.SETFCAP,
	"CAP_SETPCAP":            cap.SETPCAP,
	"CAP_SETUID":             cap.SETUID,
	"CAP_SYS_ADMIN":          cap.SYS_ADMIN,
	"CAP_SYS_BOOT":           cap.SYS_BOOT,
	"CAP_SYS_CHROOT":         cap.SYS_CHROOT,
	"CAP_SYS_MODULE":         cap.SYS_MODULE,
	"CAP_SYS_NICE":           cap.SYS_NICE,
	"CAP_SYS_PACCT":          cap.SYS_PACCT,
	"CAP_SYS_PTRACE":         cap.SYS_PTRACE,
	"CAP_SYS_RAWIO":          cap.SYS_RAWIO,
	"CAP_SYS_RESOURCE":       cap.SYS_RESOURCE,
	"CAP_SYS_TIME":           cap.SYS_TIME,
	"CAP_SYS_TTY_CONFIG":     cap.SYS_TTY_CONFIG,
	"CAP_SYSLOG":             cap.SYSLOG,
	"CAP_WAKE_ALARM":         cap.WAKE_ALARM,
}
