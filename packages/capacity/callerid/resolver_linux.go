//go:build linux

package callerid

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

// identify finds the client socket with one socket-diagnostics query, or by
// reading /proc/net/tcp{,6} when that interface is unavailable, then reads
// only the candidate processes' fd links, cgroup and executable. It never
// reads a process environment.
func (r *Resolver) identify(local, remote net.Addr) string {
	server, ok := local.(*net.TCPAddr)
	client, ok2 := remote.(*net.TCPAddr)
	if !ok || !ok2 || !client.IP.IsLoopback() {
		return Unknown
	}
	inode, uid, ok := diagSocketInode(client, server)
	if !ok {
		inode, uid, ok = clientSocketInode(filepath.Join(r.procRoot, "net"), client, server)
	}
	if !ok {
		return Unknown
	}
	if label, ok := r.remembered(inode); ok {
		return label
	}
	pid, ok := socketOwner(r.procRoot, inode, uid)
	if !ok {
		return Unknown
	}
	label := describeProcess(r.procRoot, pid)
	r.remember(inode, label)
	return label
}

// clientSocketInode returns the inode and owner of the socket whose local
// end is the client address and whose remote end is the server.
func clientSocketInode(procNet string, client, server *net.TCPAddr) (string, uint32, bool) {
	for _, table := range []string{"tcp", "tcp6"} {
		local := procSocketAddrs(client, table == "tcp6")
		remote := procSocketAddrs(server, table == "tcp6")
		if len(local) == 0 || len(remote) == 0 {
			continue
		}
		file, err := os.Open(filepath.Join(procNet, table))
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 0, 512), 1<<20)
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) < 10 || !local[fields[1]] || !remote[fields[2]] {
				continue
			}
			uid, err := strconv.ParseUint(fields[7], 10, 32)
			if err != nil || fields[9] == "0" {
				continue
			}
			_ = file.Close()
			return fields[9], uint32(uid), true
		}
		_ = file.Close()
	}
	return "", 0, false
}

// procSocketAddrs renders an address the way /proc/net/tcp{,6} prints it:
// the IP in host byte order per 32-bit word, then the port, in upper-case
// hex. A tcp6 table may hold an IPv4 peer as a v4-mapped address.
func procSocketAddrs(addr *net.TCPAddr, v6 bool) map[string]bool {
	port := fmt.Sprintf(":%04X", addr.Port)
	out := map[string]bool{}
	if ip4 := addr.IP.To4(); ip4 != nil && !v6 {
		out[fmt.Sprintf("%02X%02X%02X%02X", ip4[3], ip4[2], ip4[1], ip4[0])+port] = true
		return out
	}
	if !v6 {
		return out
	}
	ip16 := addr.IP.To16()
	if ip16 == nil {
		return out
	}
	var b strings.Builder
	for word := 0; word < 4; word++ {
		for i := 3; i >= 0; i-- {
			fmt.Fprintf(&b, "%02X", ip16[word*4+i])
		}
	}
	out[b.String()+port] = true
	return out
}

// socketOwner scans same-user processes, newest first, for the one holding
// socket inode. Short-lived callers are recent, so the scan usually stops
// after a few processes; long-lived callers are resolved once per socket and
// then remembered.
func socketOwner(proc, inode string, uid uint32) (int, bool) {
	entries, err := os.ReadDir(proc)
	if err != nil {
		return 0, false
	}
	self := os.Getpid()
	pids := make([]int, 0, len(entries))
	for _, entry := range entries {
		if pid, err := strconv.Atoi(entry.Name()); err == nil && pid > 0 && pid != self {
			pids = append(pids, pid)
		}
	}
	newestFirst(pids, lastAllocatedPID(proc))
	target := "socket:[" + inode + "]"
	for _, pid := range pids {
		dir := filepath.Join(proc, strconv.Itoa(pid))
		if info, err := os.Stat(dir); err != nil {
			continue
		} else if stat, ok := info.Sys().(*syscall.Stat_t); ok && stat.Uid != uid {
			continue
		}
		fds, err := os.ReadDir(filepath.Join(dir, "fd"))
		if err != nil {
			continue
		}
		for _, fd := range fds {
			if link, err := os.Readlink(filepath.Join(dir, "fd", fd.Name())); err == nil && link == target {
				return pid, true
			}
		}
	}
	return 0, false
}

// newestFirst orders pids from most to least recently allocated. The kernel
// hands out PIDs upward and wraps at pid_max, so after a wrap the highest
// PIDs belong to old processes: a new caller sat hundreds of places down a
// plain descending list and had exited before the scan reached it. PIDs at or
// below the last one allocated are newest, descending; the ones above it
// predate the wrap and come after. With last unknown (0) it is a plain
// descending order.
func newestFirst(pids []int, last int) {
	sort.Slice(pids, func(i, j int) bool {
		a, b := pids[i], pids[j]
		if last > 0 && (a <= last) != (b <= last) {
			return a <= last
		}
		return a > b
	})
}

// lastAllocatedPID reads the most recently allocated PID, the last field of
// /proc/loadavg, or 0 when it cannot be read.
func lastAllocatedPID(proc string) int {
	data, err := os.ReadFile(filepath.Join(proc, "loadavg"))
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return 0
	}
	pid, err := strconv.Atoi(fields[len(fields)-1])
	if err != nil || pid <= 0 {
		return 0
	}
	return pid
}

func describeProcess(proc string, pid int) string {
	dir := filepath.Join(proc, strconv.Itoa(pid))
	cgroup := ""
	if data, err := os.ReadFile(filepath.Join(dir, "cgroup")); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			// cgroup v2 is the single "0::<path>" line.
			if rest, ok := strings.CutPrefix(line, "0::"); ok {
				cgroup = rest
				break
			}
		}
	}
	executable, err := os.Readlink(filepath.Join(dir, "exe"))
	if err != nil || strings.TrimSpace(executable) == "" {
		comm, _ := os.ReadFile(filepath.Join(dir, "comm"))
		executable = strings.TrimSpace(string(comm))
	}
	return Label(cgroup, executable)
}
