//go:build darwin && cgo

package platform

/*
#include <sys/types.h>
#include <sys/stat.h>
#include <sys/acl.h>
#include <fcntl.h>
#include <membership.h>
#include <uuid/uuid.h>
#include <errno.h>
#include <string.h>

struct vrooli_broker_ace { unsigned rights; int allow; int root; int owner; };

// Inspect the ACL on an already opened inode. A missing ACL property is
// distinct from an unsupported/failed filesystem security observation.
static int vrooli_broker_acl(int fd, unsigned int owner_uid, struct vrooli_broker_ace *result, int capacity) {
 filesec_t security=filesec_init();
 if(!security) return -(errno ? errno : ENOMEM);
 struct stat st;
 if(fstatx_np(fd,&st,security)!=0) { int e=errno; filesec_free(security); return -(e ? e : EIO); }
 int present=0;
 if(filesec_query_property(security,FILESEC_ACL,&present)!=0) { int e=errno; filesec_free(security); return -(e ? e : EIO); }
 if(!present) { filesec_free(security); return 0; }
 acl_t acl=NULL;
 if(filesec_get_property(security,FILESEC_ACL,&acl)!=0 || !acl) { int e=errno; filesec_free(security); return -(e ? e : EIO); }
 filesec_free(security);
 if(acl_valid(acl)!=0) { int e=errno; acl_free(acl); return -(e ? e : EIO); }
 uuid_t root,owner;
 if(mbr_uid_to_uuid(owner_uid,owner)!=0 || mbr_uid_to_uuid(0,root)!=0) { acl_free(acl); return -EIO; }
 acl_perm_t rights[]={ACL_WRITE_DATA,ACL_APPEND_DATA,ACL_DELETE,ACL_DELETE_CHILD,ACL_WRITE_EXTATTRIBUTES,ACL_WRITE_ATTRIBUTES,ACL_WRITE_SECURITY,ACL_CHANGE_OWNER};
 int count=0, which=ACL_FIRST_ENTRY;
 for(;;) {
  acl_entry_t entry;
  errno=0;
  if(acl_get_entry(acl,which,&entry)!=0) {
   // Darwin documents EINVAL at the end of this valid kernel-supplied ACL.
   int e=errno; acl_free(acl); return e==EINVAL ? count : -(e ? e : EIO);
  }
  which=ACL_NEXT_ENTRY;
  if(count>=capacity) { acl_free(acl); return -E2BIG; }
  acl_tag_t tag;
  acl_permset_t permissions;
  if(acl_get_tag_type(entry,&tag)!=0 || acl_get_permset(entry,&permissions)!=0) { int e=errno; acl_free(acl); return -(e ? e : EIO); }
  if(tag!=ACL_EXTENDED_ALLOW && tag!=ACL_EXTENDED_DENY) { acl_free(acl); return -EINVAL; }
  result[count].allow=(tag==ACL_EXTENDED_ALLOW);
  result[count].rights=0;
  result[count].root=0; result[count].owner=0;
  for(int n=0;n<8;n++) { int value=acl_get_perm_np(permissions,rights[n]); if(value<0) { int e=errno; acl_free(acl); return -(e ? e : EIO); } if(value) result[count].rights|=(1u<<n); }
  if(result[count].allow && result[count].rights) {
   void *qualifier=acl_get_qualifier(entry);
   if(!qualifier) { int e=errno; acl_free(acl); return -(e ? e : EIO); }
   result[count].root=(memcmp(qualifier,root,sizeof(root))==0);
   result[count].owner=(memcmp(qualifier,owner,sizeof(owner))==0);
   acl_free(qualifier);
  }
  count++;
 }
}
*/
import "C"

import (
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

func darwinSystemBrokerACL(path string, expected os.FileInfo) error {
	return darwinPathACL(path, expected, 0, func(st os.FileInfo) error {
		native, ok := st.Sys().(*syscall.Stat_t)
		if !ok {
			return ErrOperationIdentity
		}
		return validateDarwinBrokerMetadata(path, st.Mode(), native.Uid, uint64(native.Nlink), expected.IsDir(), "")
	})
}

// The operator's own ACE may mutate private state; foreign users/groups may
// not. The ACL is bound to an opened inode and metadata changes invalidate the
// observation. This does not sandbox the same-UID trusted vendor.
func darwinOperatorPathACL(path string, expected os.FileInfo) error {
	owner := uint32(os.Getuid())
	return darwinPathACL(path, expected, owner, func(st os.FileInfo) error {
		native, ok := st.Sys().(*syscall.Stat_t)
		if !ok {
			return ErrOperationIdentity
		}
		original, ok := expected.Sys().(*syscall.Stat_t)
		if !ok {
			return ErrOperationIdentity
		}
		if native.Uid != original.Uid || native.Uid != 0 && native.Uid != owner || st.Mode().Perm()&0022 != 0 || st.Mode()&(os.ModeSymlink|os.ModeSetuid|os.ModeSetgid) != 0 || st.IsDir() != expected.IsDir() || !st.IsDir() && (!st.Mode().IsRegular() || native.Nlink != 1) {
			return ErrOperationIdentity
		}
		return nil
	})
}
func darwinPathACL(path string, expected os.FileInfo, owner uint32, validate func(os.FileInfo) error) error {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return err
	}
	file := os.NewFile(uintptr(fd), path)
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return err
	}
	if !os.SameFile(expected, opened) {
		return ErrOperationIdentity
	}
	if err = validate(opened); err != nil {
		return err
	}
	initial, ok := opened.Sys().(*syscall.Stat_t)
	if !ok {
		return ErrOperationIdentity
	}
	prior, ok := expected.Sys().(*syscall.Stat_t)
	if !ok || prior.Ctimespec != initial.Ctimespec {
		return ErrOperationIdentity
	}
	var entries [1024]C.struct_vrooli_broker_ace
	count := C.vrooli_broker_acl(C.int(fd), C.uint(owner), &entries[0], C.int(len(entries)))
	if count < 0 {
		return syscall.Errno(-count)
	}
	observed := make([]darwinBrokerACLGrant, int(count))
	for n := range observed {
		observed[n] = darwinBrokerACLGrant{Allow: entries[n].allow != 0, Root: entries[n].root != 0, Owner: entries[n].owner != 0, Rights: uint32(entries[n].rights)}
	}
	if owner == 0 {
		err = validateDarwinBrokerACL(observed)
	} else {
		err = validateDarwinOperatorACL(observed)
	}
	if err != nil {
		return err
	}
	final, err := file.Stat()
	if err != nil {
		return err
	}
	native, ok := final.Sys().(*syscall.Stat_t)
	if !ok || !os.SameFile(expected, final) || initial.Ctimespec != native.Ctimespec {
		return ErrOperationIdentity
	}
	return validate(final)
}
