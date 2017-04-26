// Copyright © 2015-2016 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package kexec

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"syscall"

	"github.com/platinasystems/go/internal/fit"
	"github.com/platinasystems/go/internal/flags"
	"github.com/platinasystems/go/internal/goes/lang"
	"github.com/platinasystems/go/internal/kexec"
	"github.com/platinasystems/go/internal/parms"
)

const (
	Name    = "kexec"
	Apropos = "load a new kernel for later execution"
	Usage   = "kexec [OPTIONS]..."
)

type Interface interface {
	Apropos() lang.Alt
	Main(...string) error
	String() string
	Usage() string
}

func New() Interface { return cmd{} }

type cmd struct{}

func (cmd) Apropos() lang.Alt { return apropos }

func (cmd) Main(args ...string) error {
	var err error
	flag, args := flags.New(args, "-e", "-f")
	parm, args := parms.New(args, "-c", "-i", "-k", "-l", "-x")

	if len(args) > 0 {
		return fmt.Errorf("%v: unexpected", args)
	}

	kc, err := ioutil.ReadFile("/proc/cmdline")
	if err != nil {
		fmt.Printf("Warning: unable to read kernel command line: %v\n",
			err)
	}

	kcstr := strings.TrimSpace(string(kc))

	cmdline := parm["-c"]
	if cmdline != "" {
		if cmdline[0] == '+' {
			cmdline = kcstr + " " + cmdline[1:]
		}
	} else {
		cmdline = kcstr
	}

	if image := parm["-l"]; len(image) > 0 {
		err = loadFit(image, parm["-x"])
		if err != nil {
			return err
		}
	}

	if kernel := parm["-k"]; len(kernel) > 0 {
		err = loadKernel(kernel, parm["-i"], cmdline)
		if err != nil {
			return err
		}
	}

	if flag["-e"] || flag["-f"] {
		if !flag["-f"] {
			kexec.Prepare()
		}
		err = syscall.Reboot(syscall.LINUX_REBOOT_CMD_KEXEC)
	}
	return err
}

func (cmd) String() string { return Name }
func (cmd) Usage() string  { return Usage }

func loadFit(image, x string) error {
	b, err := ioutil.ReadFile(image)
	if err != nil {
		return err
	}

	fit := fit.Parse(b)

	if len(x) == 0 {
		x = fit.DefaultConfig
	}
	config := fit.Configs[x]

	return fit.KexecLoadConfig(config, 0x0)
}

func loadKernel(kernel, initramfs, cmdline string) error {
	k, err := os.Open(kernel)
	if err != nil {
		return err
	}
	defer k.Close()

	if len(initramfs) == 0 {
		return errors.New("Initramfs (-i) must be specified")
	}

	i, err := os.Open(initramfs)
	if err != nil {
		return err
	}
	defer i.Close()

	return kexec.FileLoad(k, i, cmdline, 0)
}

var apropos = lang.Alt{
	lang.EnUS: Apropos,
}
