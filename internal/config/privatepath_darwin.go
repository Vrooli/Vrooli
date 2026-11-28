//go:build darwin && cgo

package config

/*
#include <sys/acl.h>
#include <sys/stat.h>
#include <membership.h>
#include <uuid/uuid.h>
#include <unistd.h>
#include <stdlib.h>
#include <string.h>

static int vrooli_acl_verify(int fd,uid_t owner,uid_t runtime,int has_runtime,int access,int noinherit) {
 acl_t acl=acl_get_fd_np(fd,ACL_TYPE_EXTENDED);if(!acl)return -1;
 acl_entry_t entry;int which=ACL_FIRST_ENTRY;
 while(acl_get_entry(acl,which,&entry)==0){
  which=ACL_NEXT_ENTRY;acl_tag_t tag;
  acl_flagset_t flags;if(acl_get_flagset_np(entry,&flags)!=0){acl_free(acl);return -9;}
  if(noinherit&&(acl_get_flag_np(flags,ACL_ENTRY_FILE_INHERIT)==1||acl_get_flag_np(flags,ACL_ENTRY_DIRECTORY_INHERIT)==1)){acl_free(acl);return -10;}
  if(acl_get_tag_type(entry,&tag)!=0||tag!=ACL_EXTENDED_ALLOW){acl_free(acl);return -2;}
  uuid_t *uuid=acl_get_qualifier(entry);if(!uuid){acl_free(acl);return -3;}
  id_t id=0;int type=0;int ok=mbr_uuid_to_id(*uuid,&id,&type);acl_free(uuid);
  if(ok!=0||type!=ID_TYPE_UID||(id!=owner&&(!has_runtime||id!=runtime))){acl_free(acl);return -4;}
  acl_permset_t perms;if(acl_get_permset(entry,&perms)!=0){acl_free(acl);return -5;}
  if(has_runtime&&id==runtime&&runtime!=owner&&(acl_get_perm_np(perms,ACL_WRITE_SECURITY)==1||acl_get_perm_np(perms,ACL_CHANGE_OWNER)==1)){acl_free(acl);return -6;}
  if(has_runtime&&id==runtime&&runtime!=owner&&access!=0){
   acl_perm_t writes[]={ACL_WRITE_DATA,ACL_APPEND_DATA,ACL_DELETE,ACL_DELETE_CHILD,ACL_WRITE_ATTRIBUTES,ACL_WRITE_EXTATTRIBUTES};
   for(unsigned i=0;i<sizeof(writes)/sizeof(writes[0]);i++)if(acl_get_perm_np(perms,writes[i])==1){acl_free(acl);return -7;}
   if(access==2&&(acl_get_perm_np(perms,ACL_READ_DATA)==1||acl_get_perm_np(perms,ACL_READ_EXTATTRIBUTES)==1)){acl_free(acl);return -8;}
  }
 }
 acl_free(acl);return 0;
}
static int vrooli_acl_add(acl_t *acl,uid_t uid,int owner,int access,int inherit) {
 acl_entry_t entry;if(acl_create_entry(acl,&entry)!=0)return -1;
 uuid_t uuid;if(mbr_uid_to_uuid(uid,uuid)!=0)return -2;
 if(acl_set_tag_type(entry,ACL_EXTENDED_ALLOW)!=0||acl_set_qualifier(entry,uuid)!=0)return -3;
 acl_permset_t perms;if(acl_get_permset(entry,&perms)!=0)return -4;
 acl_perm_t all[]={ACL_READ_DATA,ACL_WRITE_DATA,ACL_APPEND_DATA,ACL_EXECUTE,ACL_DELETE,ACL_DELETE_CHILD,ACL_READ_ATTRIBUTES,ACL_WRITE_ATTRIBUTES,ACL_READ_EXTATTRIBUTES,ACL_WRITE_EXTATTRIBUTES,ACL_READ_SECURITY};
 for(unsigned i=0;i<sizeof(all)/sizeof(all[0]);i++){
  if(!owner&&access!=0&&(all[i]==ACL_WRITE_DATA||all[i]==ACL_APPEND_DATA||all[i]==ACL_DELETE||all[i]==ACL_DELETE_CHILD||all[i]==ACL_WRITE_ATTRIBUTES||all[i]==ACL_WRITE_EXTATTRIBUTES))continue;
  if(!owner&&access==2&&(all[i]==ACL_READ_DATA||all[i]==ACL_READ_EXTATTRIBUTES))continue;
  if(acl_add_perm(perms,all[i])!=0)return -5;
 }
 if(owner&&(acl_add_perm(perms,ACL_WRITE_SECURITY)!=0||acl_add_perm(perms,ACL_CHANGE_OWNER)!=0))return -6;
 if(acl_set_permset(entry,perms)!=0)return -9;
 if(inherit){acl_flagset_t flags;if(acl_get_flagset_np(entry,&flags)!=0)return -7;
 if(acl_add_flag_np(flags,ACL_ENTRY_FILE_INHERIT)!=0||acl_add_flag_np(flags,ACL_ENTRY_DIRECTORY_INHERIT)!=0||acl_set_flagset_np(entry,flags)!=0)return -8;}
 return 0;
}
static int vrooli_acl_configure(int fd,uid_t owner,uid_t runtime,int has_runtime,int access,int inherit){
 acl_t acl=acl_init(2);if(!acl)return -1;
 int ret=vrooli_acl_add(&acl,owner,1,0,inherit);
 if(!ret&&has_runtime&&runtime!=owner)ret=vrooli_acl_add(&acl,runtime,0,access,inherit);
 if(!ret&&acl_set_fd_np(fd,acl,ACL_TYPE_EXTENDED)!=0)ret=-9;
 acl_free(acl);return ret;
}
*/
import "C"

import (
	"errors"
	"fmt"
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

func darwinPrivatePrincipals(p PrivatePathPolicy) (owner, runtime uint32, hasRuntime bool, err error) {
	if p.RuntimeReadOnly && p.RuntimeTraverseOnly {
		err = errors.New("ambiguous private runtime access policy")
		return
	}
	parse := func(s string) (uint32, error) {
		if !strings.HasPrefix(s, "uid:") {
			return 0, errors.New("private Darwin principal must be a numeric UID")
		}
		n, e := strconv.ParseUint(strings.TrimPrefix(s, "uid:"), 10, 32)
		if e != nil || s != fmt.Sprintf("uid:%d", n) {
			return 0, errors.New("invalid private Darwin principal")
		}
		return uint32(n), nil
	}
	owner = uint32(os.Geteuid())
	if p.OwnerPrincipal != "" {
		owner, err = parse(p.OwnerPrincipal)
		if err != nil {
			return
		}
	}
	if p.RuntimePrincipal != "" {
		runtime, err = parse(p.RuntimePrincipal)
		if err != nil {
			return
		}
		hasRuntime = true
	}
	current := uint32(os.Geteuid())
	if current != owner && (!hasRuntime || current != runtime) && current != 0 {
		err = errors.New("private Darwin path operation is outside its owner/runtime principals")
	}
	return
}
func verifyNativePrivatePath(path string, info os.FileInfo, p PrivatePathPolicy) error {
	return verifyDarwinNativeAccess(path, info, p, true)
}
func verifyNativePathAccess(path string, info os.FileInfo, p PrivatePathPolicy) error {
	return verifyDarwinNativeAccess(path, info, p, false)
}
func verifyDarwinNativeAccess(path string, info os.FileInfo, p PrivatePathPolicy, private bool) error {
	owner, runtime, hasRuntime, err := darwinPrivatePrincipals(p)
	if err != nil {
		return err
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || st == nil {
		return errors.New("private Darwin path has no native owner")
	}
	runtimeID := ""
	if hasRuntime {
		runtimeID = fmt.Sprint(runtime)
	}
	if err := verifyPrivateOwner(fmt.Sprint(st.Uid), fmt.Sprint(owner), runtimeID, "0", p); err != nil {
		return err
	}
	if private && info.Mode().Perm()&0o077 != 0 {
		return errors.New("private Darwin path has group or other permissions")
	}
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return err
	}
	file := os.NewFile(uintptr(fd), path)
	defer file.Close()
	actual, err := file.Stat()
	if err != nil || !os.SameFile(info, actual) {
		return errors.Join(errors.New("private Darwin ACL object changed"), err)
	}
	has := C.int(0)
	if hasRuntime {
		has = 1
	}
	noinherit := C.int(0)
	if p.NoInherit {
		noinherit = 1
	}
	if ret := C.vrooli_acl_verify(C.int(fd), C.uid_t(owner), C.uid_t(runtime), has, darwinPrivateAccess(p), noinherit); ret != 0 {
		return fmt.Errorf("private Darwin ACL is not qualified (%d)", int(ret))
	}
	return nil
}
func configureNativeReadOnlyPathAccess(path string, info os.FileInfo, p PrivatePathPolicy) error {
	prior := p
	prior.NoInherit = false
	if err := verifyNativePathAccess(path, info, prior); err != nil {
		return err
	}
	file, closeFile, err := openDarwinPrivateObject(path, info)
	if err != nil {
		return err
	}
	defer closeFile()
	return configurePrivateWriterAccess(file, p)
}
func ensureNativePrivateDirectory(path string, p PrivatePathPolicy) error {
	return ensureDarwinPrivateDirectoryWith(path, p, openDarwinPrivateDirectory)
}
func ensureDarwinPrivateDirectoryWith(path string, p PrivatePathPolicy, open func(string, bool) (*os.File, bool, error)) error {
	file, _, err := open(path, true)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("private Darwin directory is an alias or non-directory")
	}
	return verifyNativePrivatePath(path, info, p)
}
func ConfigurePrivateDirectoryAccess(path string, p PrivatePathPolicy) error {
	if err := EnsurePrivateDirectory(path, p); err != nil {
		return err
	}
	owner, runtime, hasRuntime, err := darwinPrivatePrincipals(p)
	if err != nil {
		return err
	}
	file, _, err := openDarwinPrivateDirectory(path, false)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	actual, err := nativePrivatePathOwner(path, info)
	if err != nil {
		return err
	}
	if actual != fmt.Sprintf("uid:%d", owner) {
		if os.Geteuid() != 0 {
			return errors.New("native private directory ownership requires explicit privileged setup")
		}
		if err = file.Chown(int(owner), -1); err != nil {
			return err
		}
	}
	has := C.int(0)
	if hasRuntime {
		has = 1
	}
	inherit := C.int(1)
	if p.NoInherit {
		inherit = 0
	}
	if ret := C.vrooli_acl_configure(C.int(file.Fd()), C.uid_t(owner), C.uid_t(runtime), has, darwinPrivateAccess(p), inherit); ret != 0 {
		return fmt.Errorf("configure exact private Darwin ACL (%d)", int(ret))
	}
	info, err = file.Stat()
	if err != nil {
		return err
	}
	return verifyNativePrivatePath(path, info, p)
}
func configurePrivateWriterAccess(file *os.File, p PrivatePathPolicy) error {
	owner, runtime, hasRuntime, err := darwinPrivatePrincipals(p)
	if err != nil {
		return err
	}
	has := C.int(0)
	if hasRuntime {
		has = 1
	}
	if ret := C.vrooli_acl_configure(C.int(file.Fd()), C.uid_t(owner), C.uid_t(runtime), has, darwinPrivateAccess(p), 0); ret != 0 {
		return fmt.Errorf("configure private temporary file ACL (%d)", int(ret))
	}
	return nil
}
func assignPrivateWriterOwner(file *os.File, p PrivatePathPolicy) error {
	if os.Geteuid() != 0 || p.OwnerPrincipal == "" {
		return nil
	}
	owner, _, _, err := darwinPrivatePrincipals(p)
	if err != nil {
		return err
	}
	return file.Chown(int(owner), -1)
}
func darwinPrivateAccess(p PrivatePathPolicy) C.int {
	if p.RuntimeTraverseOnly {
		return 2
	}
	if p.RuntimeReadOnly {
		return 1
	}
	return 0
}
func publishNativePrivateFile(from, to string) error {
	if err := os.Rename(from, to); err != nil {
		return err
	}
	return syncOwnedDirectory(filepath.Dir(to))
}
func removeNativePrivateFile(path string) error {
	if err := os.Remove(path); err != nil {
		return err
	}
	return syncOwnedDirectory(filepath.Dir(path))
}
