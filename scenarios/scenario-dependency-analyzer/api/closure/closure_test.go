package closure

import("os";"path/filepath";"testing")
func TestResolverProducesStableClosedInventory(t *testing.T){ root:=t.TempDir(); if err:=os.WriteFile(filepath.Join(root,"a.go"),[]byte("package a"),0644);err!=nil{t.Fatal(err)}; if err:=os.WriteFile(filepath.Join(root,".env"),[]byte("secret"),0600);err!=nil{t.Fatal(err)}; a,err:=NewResolver().Resolve(root,"demo","sha256:source");if err!=nil{t.Fatal(err)};b,err:=NewResolver().Resolve(root,"demo","sha256:source");if err!=nil{t.Fatal(err)};if a.ClosureDigest!=b.ClosureDigest||len(a.Files)!=1{t.Fatalf("a=%+v b=%+v",a,b)} }
