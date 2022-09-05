package driver

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/seaweedfs/seaweedfs/weed/glog"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"k8s.io/utils/mount"
)

func NewNodeServer(n *SeaweedFsDriver) *NodeServer {
	return &NodeServer{
		Driver:        n,
		volumeMutexes: NewKeyMutex(),
	}
}

func NewIdentityServer(d *SeaweedFsDriver) *IdentityServer {
	return &IdentityServer{
		Driver: d,
	}
}

func NewControllerServer(d *SeaweedFsDriver) *ControllerServer {
	return &ControllerServer{
		Driver: d,
	}
}

func NewControllerServiceCapability(cap csi.ControllerServiceCapability_RPC_Type) *csi.ControllerServiceCapability {
	return &csi.ControllerServiceCapability{
		Type: &csi.ControllerServiceCapability_Rpc{
			Rpc: &csi.ControllerServiceCapability_RPC{
				Type: cap,
			},
		},
	}
}

func ParseEndpoint(ep string) (string, string, error) {
	if strings.HasPrefix(strings.ToLower(ep), "unix://") || strings.HasPrefix(strings.ToLower(ep), "tcp://") {
		s := strings.SplitN(ep, "://", 2)
		if s[1] != "" {
			return s[0], s[1], nil
		}
	}
	return "", "", fmt.Errorf("Invalid endpoint: %v", ep)
}

func logGRPC(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	glog.V(3).Infof("GRPC %s request %+v", info.FullMethod, req)
	resp, err := handler(ctx, req)
	if err != nil {
		glog.Errorf("GRPC error: %v", err)
	}
	glog.V(3).Infof("GRPC %s response %+v", info.FullMethod, resp)
	return resp, err
}

func checkMount(targetPath string) (bool, error) {
	mounter := mount.New("")
	notMnt, err := mount.IsNotMountPoint(mounter, targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			if err = os.MkdirAll(targetPath, 0750); err != nil {
				return false, err
			}
			notMnt = true
		} else if mount.IsCorruptedMnt(err) {
			if err := mounter.Unmount(targetPath); err != nil {
				return false, err
			}
			notMnt, err = mount.IsNotMountPoint(mounter, targetPath)
		} else {
			return false, err
		}
	}
	return notMnt, nil
}

type KeyMutex struct {
	mutexes sync.Map
}

func NewKeyMutex() *KeyMutex {
	return &KeyMutex{}
}

func (km *KeyMutex) GetMutex(key string) *sync.Mutex {
	m, _ := km.mutexes.LoadOrStore(key, &sync.Mutex{})

	return m.(*sync.Mutex)
}

func (km *KeyMutex) RemoveMutex(key string) {
	km.mutexes.Delete(key)
}
