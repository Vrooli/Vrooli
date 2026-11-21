//go:build darwin && cgo

package platform

/*
#cgo LDFLAGS: -lproc
#include <libproc.h>
#include <stdint.h>
#include <sys/proc_info.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <errno.h>
#include <fcntl.h>
#include <unistd.h>
#include <sys/stat.h>
#include <sys/mman.h>
#include <stdlib.h>
#include <string.h>

// Pinned XNU proc_info_private.h wire layout (flavor 17). This private
// flavor has no SDK availability guarantee: require its complete atomic reply
// at runtime and fail closed on unsupported releases. Never substitute birth
// time, which survives exec and cannot protect image/security observations.
struct vrooli_uniqinfo {
 uint8_t uuid[16]; uint64_t uniqueid; uint64_t parent_uniqueid;
 int32_t idversion; int32_t parent_idversion; uint64_t reserve2; uint64_t reserve3;
};
_Static_assert(sizeof(struct vrooli_uniqinfo)==56,"unexpected process identity ABI");
static int vrooli_process_identity(int pid, uint64_t *uniqueid, uint32_t *version) {
 struct vrooli_uniqinfo info;
 memset(&info,0,sizeof(info)); errno=0;
 int n=proc_pidinfo(pid,17,0,&info,sizeof(info));
 if(n!=sizeof(info)) return errno ? errno : ENOTSUP;
 if(!info.uniqueid || !info.idversion) return ESTALE;
 *uniqueid=info.uniqueid; *version=(uint32_t)info.idversion; return 0;
}

// Return an open file only after its inode/device match the kernel's
// executable mapping. A replaced pathname fails rather than reading new bytes.
static int vrooli_open_image(int pid) {
 char path[PROC_PIDPATHINFO_MAXSIZE];
 if(proc_pidpath(pid,path,sizeof(path))<=0) return -(errno ? errno : EIO);
 int fd=open(path,O_RDONLY|O_CLOEXEC|O_NOFOLLOW);
 if(fd<0) return -errno;
 struct stat st;
 if(fstat(fd,&st)<0) { int e=errno; close(fd); return -e; }
 uint64_t address=0;
 for(int count=0;count<65536;count++) {
 struct proc_regionwithpathinfo region;
 int n=proc_pidinfo(pid,PROC_PIDREGIONPATHINFO,address,&region,sizeof(region));
 if(n!=sizeof(region)) break;
 if((region.prp_prinfo.pri_protection & PROT_EXEC) &&
 strcmp(region.prp_vip.vip_path,path)==0 &&
 region.prp_vip.vip_vi.vi_stat.vst_ino==st.st_ino &&
 region.prp_vip.vip_vi.vi_stat.vst_dev==st.st_dev) return fd;
 uint64_t next=region.prp_prinfo.pri_address+region.prp_prinfo.pri_size;
 if(next<=address) break;
 address=next;
 }
 close(fd); return -ESTALE;
}

static int vrooli_cwd(int pid, char *path, int length) {
 struct proc_vnodepathinfo info;
 int n = proc_pidinfo(pid, PROC_PIDVNODEPATHINFO, 0, &info, sizeof(info));
 if (n != sizeof(info)) return errno ? errno : EIO;
 size_t size = strnlen(info.pvi_cdir.vip_path, sizeof(info.pvi_cdir.vip_path));
 if (!size || size >= length) return EIO;
 memcpy(path, info.pvi_cdir.vip_path, size); path[size]=0; return 0;
}

static int vrooli_fds(int pid, void *buffer, int size) {
 errno=0;
 int n=proc_pidinfo(pid,PROC_PIDLISTFDS,0,buffer,size);
 if(n==0 && errno) return -errno;
 return n;
}

static int vrooli_socket(int pid, int fd, char *address, unsigned short *port) {
 struct socket_fdinfo info;
 int n=proc_pidfdinfo(pid,fd,PROC_PIDFDSOCKETINFO,&info,sizeof(info));
 if(n != sizeof(info)) return -(errno ? errno : EIO);
 if(info.psi.soi_kind != SOCKINFO_TCP || info.psi.soi_proto.pri_tcp.tcpsi_state != TSI_S_LISTEN) return 0;
 struct in_sockinfo *in = &info.psi.soi_proto.pri_tcp.tcpsi_ini;
 if(info.psi.soi_family == AF_INET) {
 if(!inet_ntop(AF_INET,&in->insi_laddr.ina_46.i46a_addr4,address,INET6_ADDRSTRLEN)) return -errno;
 } else if(info.psi.soi_family == AF_INET6) {
 if(!inet_ntop(AF_INET6,&in->insi_laddr.ina_6,address,INET6_ADDRSTRLEN)) return -errno;
 } else return 0;
 *port=ntohs((unsigned short)in->insi_lport); return 1;
}
*/
import "C"

import (
	"fmt"
	"golang.org/x/sys/unix"
	"os"
	"syscall"
	"unsafe"
)

func darwinWorkingDir(pid int) (string, error) {
	if pid <= 0 {
		return "", fmt.Errorf("platform: invalid pid %d", pid)
	}
	before, err := darwinObservationStamp(pid)
	if err != nil {
		return "", err
	}
	var path [4096]C.char
	if errno := C.vrooli_cwd(C.int(pid), &path[0], C.int(len(path))); errno != 0 {
		return "", syscall.Errno(errno)
	}
	after, err := darwinObservationStamp(pid)
	if err != nil || before != after {
		return "", ErrOperationIdentity
	}
	return C.GoString(&path[0]), nil
}

func nativeProcessTCPListeners(pid int) (result []ProcessTCPListener, observationErr error) {
	stamp, err := darwinObservationStamp(pid)
	if err != nil {
		return nil, err
	}
	defer func() {
		after, err := darwinObservationStamp(pid)
		if err != nil || after != stamp {
			result = nil
			observationErr = fmt.Errorf("platform: process incarnation changed during socket observation")
		}
	}()

	size := C.vrooli_fds(C.int(pid), nil, 0)
	if size < 0 {
		return nil, syscall.Errno(-size)
	}
	if size == 0 {
		return []ProcessTCPListener{}, nil
	}
	// One extra allocation margin; a full result is treated as possible truncation.
	size += C.int(32 * C.sizeof_struct_proc_fdinfo)
	if size > 16<<20 {
		return nil, fmt.Errorf("platform: process descriptor table exceeds bound")
	}
	buf := C.malloc(C.size_t(size))
	if buf == nil {
		return nil, fmt.Errorf("platform: descriptor allocation failed")
	}
	defer C.free(buf)
	n := C.vrooli_fds(C.int(pid), buf, size)
	if n < 0 {
		return nil, syscall.Errno(-n)
	}
	if n >= size || n%C.sizeof_struct_proc_fdinfo != 0 {
		return nil, fmt.Errorf("platform: unavailable or truncated descriptor table")
	}
	entries := unsafe.Slice((*C.struct_proc_fdinfo)(buf), int(n)/C.sizeof_struct_proc_fdinfo)
	for _, entry := range entries {
		if entry.proc_fdtype != C.PROX_FDTYPE_SOCKET {
			continue
		}
		var address [C.INET6_ADDRSTRLEN]C.char
		var port C.ushort
		state := C.vrooli_socket(C.int(pid), entry.proc_fd, &address[0], &port)
		if state < 0 {
			return nil, syscall.Errno(-state)
		}
		if state == 1 {
			result = append(result, ProcessTCPListener{Address: C.GoString(&address[0]), Port: uint16(port), PID: pid})
		}
	}
	return result, nil
}

func darwinOpenProcessExecutable(pid int) (*os.File, error) {
	if pid <= 0 {
		return nil, fmt.Errorf("platform: invalid pid %d", pid)
	}
	before, err := darwinObservationStamp(pid)
	if err != nil {
		return nil, err
	}
	fd := C.vrooli_open_image(C.int(pid))
	if fd < 0 {
		return nil, syscall.Errno(-fd)
	}
	file := os.NewFile(uintptr(fd), "observed-process-image")
	after, err := darwinObservationStamp(pid)
	if err != nil || before != after {
		file.Close()
		return nil, fmt.Errorf("platform: process incarnation changed during image observation")
	}
	return file, nil
}

func darwinObservationStamp(pid int) ([4]uint64, error) {
	var stamp [4]uint64
	if pid <= 0 {
		return stamp, fmt.Errorf("platform: invalid pid %d", pid)
	}
	var unique, afterUnique C.uint64_t
	var version, afterVersion C.uint32_t
	if errno := C.vrooli_process_identity(C.int(pid), &unique, &version); errno != 0 {
		return stamp, syscall.Errno(errno)
	}
	info, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err != nil {
		return stamp, err
	}
	if info == nil || int(info.Proc.P_pid) != pid || info.Proc.P_stat == darwinProcessZombie || info.Proc.P_starttime.Sec <= 0 {
		return stamp, ErrOperationIdentity
	}
	if errno := C.vrooli_process_identity(C.int(pid), &afterUnique, &afterVersion); errno != 0 {
		return stamp, syscall.Errno(errno)
	}
	if unique != afterUnique || version != afterVersion {
		return stamp, ErrOperationIdentity
	}
	return [4]uint64{uint64(info.Proc.P_starttime.Sec), uint64(info.Proc.P_starttime.Usec), uint64(unique), uint64(version)}, nil
}
