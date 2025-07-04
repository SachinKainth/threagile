package builtin

import (
	"fmt"
	"strings"

	"github.com/threagile/threagile/pkg/types"
)

type AccidentalSecretLeakRule struct{}

func NewAccidentalSecretLeakRule() *AccidentalSecretLeakRule {
	return &AccidentalSecretLeakRule{}
}

func (*AccidentalSecretLeakRule) Category() *types.RiskCategory {
	return &types.RiskCategory{
		ID:    "accidental-secret-leak",
		Title: "Accidental Secret Leak",
		Description: "Sourcecode repositories (including their histories) as well as artifact registries can accidentally contain secrets like " +
			"checked-in or packaged-in passwords, API tokens, certificates, crypto keys, etc.",
		Impact: "If this risk is unmitigated, attackers which have access to affected sourcecode repositories or artifact registries might " +
			"find secrets accidentally checked-in.",
		ASVS:       "V14 - Configuration Verification Requirements",
		CheatSheet: "https://cheatsheetseries.owasp.org/cheatsheets/Attack_Surface_Analysis_Cheat_Sheet.html",
		Action:     "Build Pipeline Hardening",
		Mitigation: "Establish measures preventing accidental check-in or package-in of secrets into sourcecode repositories " +
			"and artifact registries. This starts by using good .gitignore and .dockerignore files, but does not stop there. " +
			"See for example tools like <i>\"git-secrets\" or \"Talisman\"</i> to have check-in preventive measures for secrets. " +
			"Consider also to regularly scan your repositories for secrets accidentally checked-in using scanning tools like <i>\"gitleaks\" or \"gitrob\"</i>.",
		Check:                      "Are recommendations from the linked cheat sheet and referenced ASVS chapter applied?",
		Function:                   types.Operations,
		STRIDE:                     types.InformationDisclosure,
		DetectionLogic:             "In-scope sourcecode repositories and artifact registries.",
		RiskAssessment:             "The risk rating depends on the sensitivity of the technical asset itself and of the data assets processed.",
		FalsePositives:             "Usually no false positives.",
		ModelFailurePossibleReason: false,
		CWE:                        200,
	}
}

func (*AccidentalSecretLeakRule) SupportedTags() []string {
	// todo: how is 'nexus' being used?
	return []string{"git", "nexus"}
}

func (r *AccidentalSecretLeakRule) GenerateRisks(parsedModel *types.Model) ([]*types.Risk, error) {
	risks := make([]*types.Risk, 0)
	for _, id := range parsedModel.SortedTechnicalAssetIDs() {
		techAsset := parsedModel.TechnicalAssets[id]
		if r.skipAsset(techAsset) {
			continue
		}

		var risk *types.Risk
		if techAsset.IsTaggedWithAny("git") {
			risk = r.createRisk(parsedModel, techAsset, "Git", "Git Leak Prevention")
		} else {
			risk = r.createRisk(parsedModel, techAsset, "", "")
		}
		risks = append(risks, risk)
	}
	return risks, nil
}

func (asl AccidentalSecretLeakRule) skipAsset(techAsset *types.TechnicalAsset) bool {
	return techAsset.OutOfScope || !techAsset.Technologies.GetAttribute(types.MayContainSecrets)
}

func (r *AccidentalSecretLeakRule) createRisk(parsedModel *types.Model, technicalAsset *types.TechnicalAsset, prefix, details string) *types.Risk {
	if len(prefix) > 0 {
		prefix = " (" + prefix + ")"
	}
	title := "<b>Accidental Secret Leak" + prefix + "</b> risk at <b>" + technicalAsset.Title + "</b>"
	if len(details) > 0 {
		title += ": <u>" + details + "</u>"
	}
	impact := types.LowImpact
	highestProcessedConfidentiality := parsedModel.HighestProcessedConfidentiality(technicalAsset)
	highestProcessedIntegrity := parsedModel.HighestProcessedIntegrity(technicalAsset)
	highestProcessedAvailability := parsedModel.HighestProcessedAvailability(technicalAsset)
	if highestProcessedConfidentiality >= types.Confidential ||
		highestProcessedIntegrity >= types.Critical ||
		highestProcessedAvailability >= types.Critical {
		impact = types.MediumImpact
	}
	if highestProcessedConfidentiality == types.StrictlyConfidential ||
		highestProcessedIntegrity == types.MissionCritical ||
		highestProcessedAvailability == types.MissionCritical {
		impact = types.HighImpact
	}
	// create risk
	risk := &types.Risk{
		CategoryId:                   r.Category().ID,
		Severity:                     types.CalculateSeverity(types.Unlikely, impact),
		ExploitationLikelihood:       types.Unlikely,
		ExploitationImpact:           impact,
		Title:                        title,
		MostRelevantTechnicalAssetId: technicalAsset.Id,
		DataBreachProbability:        types.Probable,
		DataBreachTechnicalAssetIDs:  []string{technicalAsset.Id},
	}
	risk.SyntheticId = risk.CategoryId + "@" + technicalAsset.Id
	return risk
}

func (r *AccidentalSecretLeakRule) MatchRisk(parsedModel *types.Model, risk string) bool {
	categoryId := r.Category().ID
	for _, id := range parsedModel.SortedTechnicalAssetIDs() {
		techAsset := parsedModel.TechnicalAssets[id]
		if strings.EqualFold(risk, categoryId+"@"+techAsset.Id) || strings.EqualFold(risk, categoryId+"@*") {
			return true
		}
	}

	return false
}

func (r *AccidentalSecretLeakRule) ExplainRisk(parsedModel *types.Model, risk string) []string {
	categoryId := r.Category().ID
	explanation := make([]string, 0)
	for _, id := range parsedModel.SortedTechnicalAssetIDs() {
		techAsset := parsedModel.TechnicalAssets[id]
		if strings.EqualFold(risk, categoryId+"@"+techAsset.Id) || strings.EqualFold(risk, categoryId+"@*") {
			if !techAsset.OutOfScope && (techAsset.Technologies.GetAttribute(types.SourcecodeRepository) || techAsset.Technologies.GetAttribute(types.ArtifactRegistry)) {
				riskExplanation := r.explainRisk(parsedModel, techAsset)
				if riskExplanation != nil {
					if len(explanation) > 0 {
						explanation = append(explanation, "")
					}

					explanation = append(explanation, []string{
						fmt.Sprintf("technical asset %q", techAsset.Id),
						fmt.Sprintf("  - out of scope: %v (=false)", techAsset.OutOfScope),
						fmt.Sprintf("  - technology: %v (has either [%q, %q])", techAsset.Technologies.String(), types.SourcecodeRepository, types.ArtifactRegistry),
					}...)

					if techAsset.IsTaggedWithAny("git") {
						explanation = append(explanation, "  is tagged with 'git'")
					}

					explanation = append(explanation, riskExplanation...)
				}
			}
		}
	}

	return explanation
}

func (r *AccidentalSecretLeakRule) explainRisk(parsedModel *types.Model, technicalAsset *types.TechnicalAsset) []string {
	explanation := make([]string, 0)
	impact := types.LowImpact
	if parsedModel.HighestProcessedConfidentiality(technicalAsset) == types.StrictlyConfidential ||
		parsedModel.HighestProcessedIntegrity(technicalAsset) == types.MissionCritical ||
		parsedModel.HighestProcessedAvailability(technicalAsset) == types.MissionCritical {
		impact = types.HighImpact

		explanation = append(explanation,
			fmt.Sprintf("    - impact is %v because", impact),
		)

		if parsedModel.HighestProcessedConfidentiality(technicalAsset) == types.StrictlyConfidential {
			explanation = append(explanation,
				fmt.Sprintf("      => highest confidentiality: %v (==%v)", parsedModel.HighestProcessedConfidentiality(technicalAsset), types.StrictlyConfidential),
			)
		}

		if parsedModel.HighestProcessedIntegrity(technicalAsset) == types.MissionCritical {
			explanation = append(explanation,
				fmt.Sprintf("      => highest integrity: %v (==%v)", parsedModel.HighestProcessedIntegrity(technicalAsset), types.MissionCritical),
			)
		}

		if parsedModel.HighestProcessedAvailability(technicalAsset) == types.MissionCritical {
			explanation = append(explanation,
				fmt.Sprintf("      => highest availability: %v (==%v)", parsedModel.HighestProcessedAvailability(technicalAsset), types.MissionCritical),
			)
		}
	} else if parsedModel.HighestProcessedConfidentiality(technicalAsset) >= types.Confidential ||
		parsedModel.HighestProcessedIntegrity(technicalAsset) >= types.Critical ||
		parsedModel.HighestProcessedAvailability(technicalAsset) >= types.Critical {
		impact = types.MediumImpact
		explanation = append(explanation,
			fmt.Sprintf("    - impact is %v because", impact),
		)

		if parsedModel.HighestProcessedConfidentiality(technicalAsset) == types.StrictlyConfidential {
			explanation = append(explanation,
				fmt.Sprintf("     =>  highest confidentiality: %v (>=%v)", parsedModel.HighestProcessedConfidentiality(technicalAsset), types.Confidential),
			)
		}

		if parsedModel.HighestProcessedIntegrity(technicalAsset) == types.MissionCritical {
			explanation = append(explanation,
				fmt.Sprintf("     =>  highest integrity: %v (==%v)", parsedModel.HighestProcessedIntegrity(technicalAsset), types.Critical),
			)
		}

		if parsedModel.HighestProcessedAvailability(technicalAsset) == types.MissionCritical {
			explanation = append(explanation,
				fmt.Sprintf("     =>  highest availability: %v (==%v)", parsedModel.HighestProcessedAvailability(technicalAsset), types.Critical),
			)
		}
	} else {
		explanation = append(explanation,
			fmt.Sprintf("     - impact is %v (default)", impact),
		)
	}

	return explanation
}
