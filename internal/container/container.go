package container

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/containerd/cgroups/v3/cgroup2"
)

const initEnv = "GOCKER_INIT"

// Run initializes the container environment and starts the init process.
// If the environment variable GOCKER_INIT is set to "1", it runs the init process
// inside the container with the specified root filesystem and command.
// If the environment variable is not set, it starts a new process with the same executable
// and passes the environment variable to indicate that it is the init process.
// It also attaches the process to a cgroup for resource management.
func Run(rootfs string, command []string) error {
	if os.Getenv(initEnv) == "1" {
		return startInitProcess(rootfs, command)
	}

	cmd := exec.Command("/proc/self/exe")
	cmd.Args = append([]string{"/proc/self/exe"}, os.Args[1:]...)
	cmd.Env = append(os.Environ(), initEnv+"=1")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	err = attachToCgroup(uint64(cmd.Process.Pid))
	if err != nil {
		_ = cmd.Process.Kill()
		return fmt.Errorf("failed to apply cgroup: %w", err)
	}

	return cmd.Wait()
}

// startInitProcess sets up the container environment by changing the root filesystem,
// changing the working directory, mounting the proc filesystem, and executing the entrypoint script or command.
// It is called when the GOCKER_INIT environment variable is set to "1".
// It expects the root filesystem to be already set up and the command to be executed inside the container.
// The entrypoint script is expected to be located at /usr/local/bin/docker-entrypoint.sh
// If the script does not exist, it falls back to executing the first command in the command slice.
// The command slice is expected to contain the command and its arguments to be executed inside the container
func startInitProcess(rootfs string, command []string) error {
	err := syscall.Chroot(rootfs)
	if err != nil {
		return fmt.Errorf("chroot failed: %w", err)
	}
	err = syscall.Chdir("/")
	if err != nil {
		return fmt.Errorf("chdir failed: %w", err)
	}
	err = syscall.Mount("proc", "/proc", "proc", 0, "")
	if err != nil {
		return fmt.Errorf("mount /proc failed: %w", err)
	}

	entrypoint := command[0]
	_, err = os.Stat(entrypoint) // Ensure the command exists
	if os.IsNotExist(err) {
		return fmt.Errorf("entrypoint command does not exist: %s", entrypoint)
	}
	command = command[1:]

	fmt.Println("Running entrypoint:", entrypoint)

	args := append([]string{entrypoint}, command...)

	return syscall.Exec(entrypoint, args, os.Environ())
}

// attachToCgroup attaches the current process to a cgroup with specified resource limits.
// It creates a new cgroup with memory and CPU limits, adds the process to the cgroup,
// and returns an error if any operation fails.
func attachToCgroup(pid uint64) error {
	maxMemory := int64(1024 * 1024 * 1024)
	quota := int64(10000)
	period := uint64(100000)
	res := cgroup2.Resources{
		Memory: &cgroup2.Memory{
			Max: &maxMemory,
		},
		CPU: &cgroup2.CPU{
			Max: cgroup2.NewCPUMax(&quota, &period),
		},
	}

	cg, err := cgroup2.NewSystemd("/", "gocker-container.slice", -1, &res)
	if err != nil {
		return fmt.Errorf("failed to create cgroup: %w", err)
	}

	if err := cg.AddProc(pid); err != nil {
		return fmt.Errorf("failed to add process to cgroup: %w", err)
	}

	return nil
}
