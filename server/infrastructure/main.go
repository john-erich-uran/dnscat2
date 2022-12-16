package main

import (
	"fmt"
	"time"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi-command/sdk/go/command/local"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi-tls/sdk/v4/go/tls"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		sg, err := ec2.NewSecurityGroup(ctx, "ec2-runner-sg", &ec2.SecurityGroupArgs{
			Description: pulumi.String("SSH and DNS"),
			Ingress: ec2.SecurityGroupIngressArray{
				&ec2.SecurityGroupIngressArgs{
					Description: pulumi.String("SSH"),
					FromPort:    pulumi.Int(22),
					ToPort:      pulumi.Int(22),
					Protocol:    pulumi.String("tcp"),
					CidrBlocks: pulumi.StringArray{
						pulumi.String("0.0.0.0/0"),
					},
				},
				&ec2.SecurityGroupIngressArgs{
					Description: pulumi.String("DNS UDP"),
					FromPort:    pulumi.Int(53),
					ToPort:      pulumi.Int(53),
					Protocol:    pulumi.String("udp"),
					CidrBlocks: pulumi.StringArray{
						pulumi.String("0.0.0.0/0"),
					},
				},
				&ec2.SecurityGroupIngressArgs{
					Description: pulumi.String("DNS TCP"),
					FromPort:    pulumi.Int(53),
					ToPort:      pulumi.Int(53),
					Protocol:    pulumi.String("tcp"),
					CidrBlocks: pulumi.StringArray{
						pulumi.String("0.0.0.0/0"),
					},
				},
			},
			Egress: ec2.SecurityGroupEgressArray{
				&ec2.SecurityGroupEgressArgs{
					FromPort: pulumi.Int(0),
					ToPort:   pulumi.Int(0),
					Protocol: pulumi.String("-1"),
					CidrBlocks: pulumi.StringArray{
						pulumi.String("0.0.0.0/0"),
					},
					Ipv6CidrBlocks: pulumi.StringArray{
						pulumi.String("::/0"),
					},
				},
			},
		})
		if err != nil {
			return fmt.Errorf("ec2-runner-sg: %w", err)
		}

		privateKey, err := tls.NewPrivateKey(ctx, "tls-private-key", &tls.PrivateKeyArgs{
			Algorithm: pulumi.String("RSA"),
			RsaBits:   pulumi.Int(4096),
		})
		if err != nil {
			return fmt.Errorf("tls-private-key: %w", err)
		}

		keyPair, err := ec2.NewKeyPair(ctx, "ec2-runner-key", &ec2.KeyPairArgs{
			PublicKey: privateKey.PublicKeyOpenssh,
		})
		if err != nil {
			return fmt.Errorf("new key pair: %w", err)
		}

		runner, err := ec2.NewInstance(ctx, "ec2-runner", &ec2.InstanceArgs{
			InstanceType:        pulumi.String("t2.micro"),
			VpcSecurityGroupIds: pulumi.StringArray{sg.ID()},
			Ami:                 pulumi.String("ami-0b0dcb5067f052a63"),
			KeyName:             keyPair.KeyName,
		})
		if err != nil {
			return fmt.Errorf("ec2-runner: %w", err)
		}

		alwaysTrigger := pulumi.Array{
			pulumi.String(fmt.Sprintf("%d", time.Now().UnixMilli())),
		}

		tarCmd, err := local.NewCommand(
			ctx,
			"tar-cmd",
			&local.CommandArgs{
				Dir:      pulumi.String("../"),
				Create:   pulumi.String("rm -rf ./infrastructure/server.tar.gz && tar zcfv ./infrastructure/server.tar.gz --exclude='.bundle' --exclude='infrastructure' ."),
				Triggers: alwaysTrigger,
			},
			pulumi.DependsOn([]pulumi.Resource{runner}),
		)
		if err != nil {
			return fmt.Errorf("tar-cmd: %w", err)
		}

		connection := remote.ConnectionArgs{
			Host:       runner.PublicIp,
			PrivateKey: privateKey.PrivateKeyOpenssh,
			User:       pulumi.String("ec2-user"),
		}

		uploadServerCmd, err := remote.NewCopyFile(
			ctx,
			"upload-server-cmd",
			&remote.CopyFileArgs{
				Connection: connection,
				LocalPath:  pulumi.String("./server.tar.gz"),
				RemotePath: pulumi.String("/home/ec2-user/server.tar.gz"),
				Triggers:   alwaysTrigger,
			},
			pulumi.DependsOn([]pulumi.Resource{tarCmd}),
		)
		if err != nil {
			return fmt.Errorf("upload-server-cmd: %w", err)
		}

		installServerCmd, err := remote.NewCommand(
			ctx,
			"install-server-cmd",
			&remote.CommandArgs{
				Connection: connection,
				Create:     pulumi.String("rm -rf server && mkdir -p server && tar xzf server.tar.gz -C server && bash ./server/install.sh root"),
				Triggers:   alwaysTrigger,
			},
			pulumi.DependsOn([]pulumi.Resource{uploadServerCmd}),
		)
		if err != nil {
			return fmt.Errorf("install-server-cmd: %w", err)
		}

		ctx.Export("ip", runner.PublicIp)
		ctx.Export("sshKey", privateKey.PrivateKeyOpenssh)
		ctx.Export("installStdout", installServerCmd.Stdout)
		ctx.Export("installStderr", installServerCmd.Stderr)
		return nil
	})
}
