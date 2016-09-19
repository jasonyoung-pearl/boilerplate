package config

import (
	"fmt"
	"github.com/gruntwork-io/boilerplate/util"
	"github.com/gruntwork-io/boilerplate/errors"
	"github.com/gruntwork-io/boilerplate/variables"
)

// Get a value for each of the variables specified in boilerplateConfig, other than those already in existingVariables.
// The value for a variable can come from the user (if the  non-interactive option isn't set), the default value in the
// config, or a command line option.
func GetVariables(options *BoilerplateOptions, boilerplateConfig *BoilerplateConfig) (map[string]interface{}, error) {
	vars := map[string]interface{}{}
	for key, value := range options.Vars {
		vars[key] = value
	}

	variablesInConfig := getAllVariablesInConfig(boilerplateConfig)

	for _, variable := range variablesInConfig {
		var value interface{}
		var err error

		value, alreadyExists := vars[variable.Name()]
		if !alreadyExists {
			value, err = getVariable(variable, options)
			if err != nil {
				return vars, err
			}
		}

		unmarshalled, err := variables.UnmarshalValueForVariable(value, variable)
		if err != nil {
			return vars, err
		}

		vars[variable.Name()] = unmarshalled
	}

	return vars, nil
}

// Get all the variables defined in the given config and its dependencies
func getAllVariablesInConfig(boilerplateConfig *BoilerplateConfig) []variables.Variable {
	allVariables := []variables.Variable{}

	allVariables = append(allVariables, boilerplateConfig.Variables...)

	for _, dependency := range boilerplateConfig.Dependencies {
		allVariables = append(allVariables, dependency.GetNamespacedVariables()...)
	}

	return allVariables
}

// Get a value for the given variable. The value can come from the user (if the non-interactive option isn't set), the
// default value in the config, or a command line option.
func getVariable(variable variables.Variable, options *BoilerplateOptions) (interface{}, error) {
	valueFromVars, valueSpecifiedInVars := getVariableFromVars(variable, options)

	if valueSpecifiedInVars {
		util.Logger.Printf("Using value specified via command line options for variable '%s': %s", variable.FullName(), valueFromVars)
		return valueFromVars, nil
	} else if options.NonInteractive && variable.Default() != nil {
		util.Logger.Printf("Using default value for variable '%s': %v", variable.FullName(), variable.Default())
		return variable.Default(), nil
	} else if options.NonInteractive {
		return nil, errors.WithStackTrace(MissingVariableWithNonInteractiveMode(variable.FullName()))
	} else {
		return getVariableFromUser(variable, options)
	}
}

// Return the value of the given variable from vars passed in as command line options
func getVariableFromVars(variable variables.Variable, options *BoilerplateOptions) (interface{}, bool) {
	for name, value := range options.Vars {
		if name == variable.Name() {
			return value, true
		}
	}

	return nil, false
}

// Get the value for the given variable by prompting the user
func getVariableFromUser(variable variables.Variable, options *BoilerplateOptions) (interface{}, error) {
	util.BRIGHT_GREEN.Printf("\n%s\n", variable.FullName())
	if variable.Description() != "" {
		fmt.Printf("  %s\n", variable.Description())
	}
	if variable.Default() != nil {
		fmt.Printf("  (default: %s)\n", variable.Default())
	}
	fmt.Printf("  (type: %s, example: %s)\n", variable.Type(), variable.Type().Example())
	fmt.Println()

	value, err := util.PromptUserForInput("  Enter a value")
	if err != nil {
		return "", err
	}

	if value == "" {
		// TODO: what if the user wanted an empty string instead of the default?
		util.Logger.Printf("Using default value for variable '%s': %v", variable.FullName(), variable.Default())
		return variable.Default(), nil
	}

	return variables.ParseYamlString(value)
}

// Custom error types

type MissingVariableWithNonInteractiveMode string
func (variableName MissingVariableWithNonInteractiveMode) Error() string {
	return fmt.Sprintf("Variable '%s' does not have a default, no value was specified at the command line using the --%s option, and the --%s flag is set, so cannot prompt user for a value.", string(variableName), OPT_VAR, OPT_NON_INTERACTIVE)
}