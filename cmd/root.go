package cmd

import (
	"fmt"
	"github.com/4nte/protodist/git"
	"os"
	"strings"

	"github.com/4nte/protodist/internal/distribute"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/spf13/viper"
)

const (
	envPrefix             = "PROTODIST"
	defaultConfigFilename = "protodist"
)

var (
	gitRef       string
	gitHost      string
	gitRepoOwner string
	gitToken     string
	protoOutDir  string
	deploy       string
	deployDir    string
	verbose      bool
	dryRun       bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "protodist",
	Short: "Distribute protobuf packages via GIT",
	Long:  `Distribute protobuf packages via GIT`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// You can bind cobra and viper in a few locations, but PersistencePreRunE on the root command works well
		return initializeConfig(cmd)
	},
	Run: func(cmd *cobra.Command, args []string) {
		if deploy == "git" {
			if gitRepoOwner == "" {
				panic("PROTODIST_GIT_REPO_OWNER must be set")
			}

			if gitHost == "" {
				panic("PROTODIST_GIT_HOST must be set")
			}

			if gitRef == "" {
				panic("PROTODIST_GIT_REF must be set")
			}

			if gitToken == "" {
				fmt.Println("warning: PROTODIST_GIT_TOKEN is not set, protodist can access only public repos now")
			}
		} else if deploy == "local" {
			if deployDir == "" {
				panic("PROTODIST_DEPLOY_DIR must be set when deploy strategy is 'local'")
			}
			// TODO: This shouldn't be hardcoded
			gitRepoOwner = "spotsie"
			gitHost = "github.com"
			gitRef = "refs/heads/local"
		}

		gitCfg, err := git.NewConfig(gitRepoOwner, gitHost, gitRef, gitToken)
		if err != nil {
			panic(err)
		}

		distribute.Distribute(gitCfg, protoOutDir, dryRun, deploy, deployDir)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	//cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// TODO: refactor to use git_ref instead and then infer if the ref is branch or tag
	rootCmd.PersistentFlags().StringVar(&gitRepoOwner, "git_repo_owner", "", "git repo owner")
	rootCmd.PersistentFlags().StringVar(&gitRef, "git_ref", "", "git ref (e.g refs/heads/foo-branch, refs/tags/foo-tag)")
	rootCmd.PersistentFlags().StringVar(&gitHost, "git_host", "", "git host")
	rootCmd.PersistentFlags().StringVar(&gitToken, "git_token", "", "git token")
	rootCmd.PersistentFlags().StringVar(&protoOutDir, "proto_out_dir", "", "proto output directory")
	rootCmd.PersistentFlags().StringVar(&deploy, "deploy", "git", "deploy to: git or local")
	rootCmd.PersistentFlags().StringVar(&deployDir, "deploy_dir", "", "local deploy directory")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "show verbose logs")
	rootCmd.PersistentFlags().BoolVarP(&dryRun, "dry_run", "d", false, "don't git push")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	if err := viper.BindPFlags(rootCmd.Flags()); err != nil {
		panic(err)
	}
}
func bindFlags(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		// Environment variables can't have dashes in them, so bind them to their equivalent
		// keys with underscores, e.g. --favorite-color to STING_FAVORITE_COLOR
		if strings.Contains(f.Name, "-") {
			envVarSuffix := strings.ToUpper(strings.ReplaceAll(f.Name, "-", "_"))
			err := v.BindEnv(f.Name, fmt.Sprintf("%s_%s", envPrefix, envVarSuffix))
			if err != nil {
				panic(err)
			}
		}

		// Apply the viper config value to the flag when the flag is not set and viper has a value
		if !f.Changed && v.IsSet(f.Name) {
			val := v.Get(f.Name)
			err := cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
			if err != nil {
				panic(err)
			}
		}
	})
}
func initializeConfig(cmd *cobra.Command) error {
	v := viper.New()

	// Set the base name of the config file, without the file extension.
	v.SetConfigName(defaultConfigFilename)

	// Set as many paths as you like where viper should look for the
	// config file. We are only looking in the current working directory.
	v.AddConfigPath(".")

	// Attempt to read the config file, gracefully ignoring errors
	// caused by a config file not being found. Return an error
	// if we cannot parse the config file.
	if err := v.ReadInConfig(); err != nil {
		// It's okay if there isn't a config file
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	// When we bind flags to environment variables expect that the
	// environment variables are prefixed, e.g. a flag like --number
	// binds to an environment variable STING_NUMBER. This helps
	// avoid conflicts.
	v.SetEnvPrefix(envPrefix)

	// Bind to environment variables
	// Works great for simple config names, but needs help for names
	// like --favorite-color which we fix in the bindFlags function
	v.AutomaticEnv()

	// Bind the current command's flags to viper
	bindFlags(cmd, v)

	return nil
}
