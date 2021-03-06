package fs

import (
	"os"
	"os/user"
	"path/filepath"
	"sync"
	"strconv"

	"bazil.org/fuse"
	fusefs "bazil.org/fuse/fs"
	_ "bazil.org/fuse/fs/fstestutil"

	"github.com/opencontainers/runc/libcontainer/cgroups"

	"golang.org/x/net/context"
)

const (
	_ = iota
	INODE_DIR
	INODE_HELLO
	INODE_MEMINFO
	INODE_DISKSTATS
	INODE_CPUINFO
	INODE_STAT
	INODE_NET_DEV
)

var (
	fileMap = make(map[string]FileInfo)

	direntsOnce sync.Once
	dirents     []fuse.Dirent
)

// Dir implements both Node and Handle for the root directory.
type Dir struct {
	cgroupdir string
	vethName  string
}

type FileInfo struct {
	initFunc   func(cgroupdir string) fusefs.Node
	inode      uint64
	subsysName string
}

func (Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = INODE_DIR

	user, err := user.Current()
	uid, err := strconv.ParseInt(user.Uid, 10, 32)
	if err != nil {
		panic(err)
	}
	gid, err := strconv.ParseInt(user.Gid, 10, 32)
	if err != nil {
		panic(err)
	}
	a.Uid = uint32(uid)
	a.Gid = uint32(gid)
	a.Mode = os.ModeDir | 0777
	return nil
}

func (d Dir) Lookup(ctx context.Context, name string) (fusefs.Node, error) {
	if name == "hello" {
		return File{}, nil
	} else if fileInfo, ok := fileMap[name]; ok {
		var path string

		if len(fileInfo.subsysName) == 0 {
			path = d.vethName
		} else {
			mountPoint, err := cgroups.FindCgroupMountpoint(fileInfo.subsysName)
			if err != nil {
				return nil, fuse.ENODATA
			}
			path = filepath.Join(mountPoint, d.cgroupdir)
		}

		return fileInfo.initFunc(path), nil
	}
	return nil, fuse.ENOENT
}

func (Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	direntsOnce.Do(func() {
		dirents = append(dirents, fuse.Dirent{Inode: INODE_HELLO, Name: "hello", Type: fuse.DT_File})
		for k, v := range fileMap {
			dirents = append(dirents, fuse.Dirent{Inode: v.inode, Name: k, Type: fuse.DT_File})
		}
	})
	return dirents, nil
}
