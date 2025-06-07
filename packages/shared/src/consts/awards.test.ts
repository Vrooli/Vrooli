import { describe, expect, it } from "vitest";
import { AwardCategory } from "../api/types.js";
import { awardNames, awardVariants } from "./awards.js";

describe("awards constants", () => {
    describe("awardVariants", () => {
        it("should have correct variant arrays for each award category", () => {
            expect(awardVariants.Streak).toEqual([7, 30, 100, 200, 365, 500, 750, 1000]);
            expect(awardVariants.Reputation).toEqual([10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000]);
            expect(awardVariants.ObjectBookmark).toEqual([1, 100, 500]);
            expect(awardVariants.ObjectReact).toEqual([1, 100, 1000, 10000]);
            expect(awardVariants.PullRequestCreate).toEqual([1, 5, 10, 25, 50, 100, 250, 500]);
            expect(awardVariants.PullRequestComplete).toEqual([1, 5, 10, 25, 50, 100, 250, 500]);
            expect(awardVariants.ApiCreate).toEqual([1, 5, 10, 25, 50]);
            expect(awardVariants.SmartContractCreate).toEqual([1, 5, 10, 25]);
            expect(awardVariants.CommentCreate).toEqual([1, 5, 10, 25, 50, 100, 250, 500, 1000]);
            expect(awardVariants.IssueCreate).toEqual([1, 5, 10, 25, 50, 100, 250]);
            expect(awardVariants.NoteCreate).toEqual([1, 5, 10, 25, 50, 100]);
            expect(awardVariants.ProjectCreate).toEqual([1, 5, 10, 25, 50, 100]);
            expect(awardVariants.ReportEnd).toEqual([1, 5, 10, 25, 50, 100]);
            expect(awardVariants.ReportContribute).toEqual([1, 5, 10, 25, 50, 100, 250, 500, 1000]);
            expect(awardVariants.RunRoutine).toEqual([1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000]);
            expect(awardVariants.RunProject).toEqual([1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 10000]);
            expect(awardVariants.RoutineCreate).toEqual([1, 5, 10, 25, 50, 100, 250, 500, 1000]);
            expect(awardVariants.StandardCreate).toEqual([1, 5, 10, 25, 50]);
            expect(awardVariants.OrganizationCreate).toEqual([1, 2, 5, 10]);
            expect(awardVariants.OrganizationJoin).toEqual([1, 5, 10, 25]);
            expect(awardVariants.UserInvite).toEqual([1, 5, 10, 25, 50, 100]);
        });

        it("should have ascending ordered arrays", () => {
            Object.values(awardVariants).forEach(variants => {
                for (let i = 1; i < variants.length; i++) {
                    expect(variants[i]).toBeGreaterThan(variants[i - 1]);
                }
            });
        });

        it("should include all award categories except special cases", () => {
            const variantKeys = Object.keys(awardVariants);
            const allCategories = Object.values(AwardCategory);
            // AccountAnniversary and AccountNew are special cases
            const expectedKeys = allCategories.filter(cat =>
                cat !== "AccountAnniversary" &&
                cat !== "AccountNew"
            );

            expect(variantKeys.sort()).toEqual(expectedKeys.sort());
        });
    });

    describe("awardNames", () => {
        describe("AccountAnniversary", () => {
            it("should return correct award for years", () => {
                const result = awardNames.AccountAnniversary(2);
                expect(result.name).toBe("AccountAnniversary_Title");
                expect(result.nameVariables).toEqual({ count: 2 });
                expect(result.body).toBe("AccountAnniversary_Body");
                expect(result.bodyVariables).toEqual({ count: 2 });
                expect(result.level).toBe(2);
            });

            it("should return next tier when findNext is true", () => {
                const result = awardNames.AccountAnniversary(2, true);
                expect(result.nameVariables).toEqual({ count: 3 });
                expect(result.bodyVariables).toEqual({ count: 3 });
                expect(result.level).toBe(3);
            });
        });

        describe("AccountNew", () => {
            it("should return correct award for new accounts", () => {
                const result = awardNames.AccountNew(0);
                expect(result.name).toBe("AccountNew_Title");
                expect(result.body).toBe("AccountNew_Body");
                expect(result.level).toBe(0);
                expect(result.nameVariables).toBeUndefined();
                expect(result.bodyVariables).toBeUndefined();
            });
        });

        describe("ApiCreate", () => {
            it("should return correct award based on awardTier logic", () => {
                // Based on awardVariants.ApiCreate: [1, 5, 10, 25, 50]
                // The awardTier function finds the next tier when count < tier
                const result0 = awardNames.ApiCreate(0);
                expect(result0.name).toBe("ApiCreate1_Title"); // 0 < 1, so returns tier 1

                const result1 = awardNames.ApiCreate(1);
                expect(result1.name).toBe("ApiCreate5_Title"); // 1 < 5, so returns tier 5
                expect(result1.level).toBe(5);

                const result5 = awardNames.ApiCreate(5);
                expect(result5.name).toBe("ApiCreate10_Title"); // 5 < 10, so returns tier 10
                expect(result5.level).toBe(10);

                const result50 = awardNames.ApiCreate(50);
                expect(result50.name).toBe("ApiCreate50_Title"); // 50 is highest tier
                expect(result50.level).toBe(50);

                const result100 = awardNames.ApiCreate(100);
                expect(result100.name).toBe("ApiCreate50_Title"); // Beyond max, returns highest
                expect(result100.level).toBe(50);
            });

            it("should handle findNext parameter", () => {
                const result = awardNames.ApiCreate(1, true);
                expect(result.level).toBe(10); // Next after 5 is 10

                const resultNext = awardNames.ApiCreate(25, true);
                expect(resultNext.level).toBe(50); // Next after 25 is 50
            });
        });

        describe("SmartContractCreate", () => {
            it("should return correct award based on awardTier logic", () => {
                // Based on awardVariants.SmartContractCreate: [1, 5, 10, 25]
                const result1 = awardNames.SmartContractCreate(1);
                expect(result1.name).toBe("SmartContractCreate5_Title");
                expect(result1.body).toBe("SmartContractCreate_Body");
                expect(result1.level).toBe(5);

                const result25 = awardNames.SmartContractCreate(25);
                expect(result25.name).toBe("SmartContractCreate25_Title");
                expect(result25.level).toBe(25);
            });

            it("should handle counts for first tier", () => {
                const result = awardNames.SmartContractCreate(0);
                expect(result.name).toBe("SmartContractCreate1_Title");
                expect(result.level).toBe(1);
            });
        });

        describe("CommentCreate", () => {
            it("should return correct award based on awardTier logic", () => {
                const result10 = awardNames.CommentCreate(10);
                expect(result10.name).toBe("CommentCreate25_Title"); // 10 < 25
                expect(result10.body).toBe("CommentCreate_Body");
                expect(result10.level).toBe(25);

                const result1000 = awardNames.CommentCreate(1000);
                expect(result1000.name).toBe("CommentCreate1000_Title");
                expect(result1000.level).toBe(1000);
            });
        });

        describe("ObjectBookmark", () => {
            it("should return correct award for bookmark counts", () => {
                // Based on awardVariants.ObjectBookmark: [1, 100, 500]
                const result1 = awardNames.ObjectBookmark(1);
                expect(result1.name).toBe("ObjectBookmark100_Title"); // 1 < 100
                expect(result1.body).toBe("ObjectBookmark_Body");
                expect(result1.level).toBe(100);

                const result500 = awardNames.ObjectBookmark(500);
                expect(result500.name).toBe("ObjectBookmark500_Title");
                expect(result500.level).toBe(500);
            });
        });

        describe("ObjectReact", () => {
            it("should return correct award for reaction counts", () => {
                // Based on awardVariants.ObjectReact: [1, 100, 1000, 10000]
                const result100 = awardNames.ObjectReact(100);
                expect(result100.name).toBe("ObjectReact1000_Title"); // 100 < 1000
                expect(result100.body).toBe("ObjectReact_Body");
                expect(result100.level).toBe(1000);

                const result10000 = awardNames.ObjectReact(10000);
                expect(result10000.name).toBe("ObjectReact10000_Title");
                expect(result10000.level).toBe(10000);
            });
        });

        describe("Reputation", () => {
            it("should return correct award for reputation points", () => {
                // Based on awardVariants.Reputation: [10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000]
                const result100 = awardNames.Reputation(100);
                expect(result100.name).toBe("ReputationPoints250_Title"); // 100 < 250
                expect(result100.body).toBe("ReputationPoints_Body");
                expect(result100.level).toBe(250);

                const result10000 = awardNames.Reputation(10000);
                expect(result10000.name).toBe("ReputationPoints10000_Title");
                expect(result10000.level).toBe(10000);
            });

            it("should handle reputation with findNext", () => {
                const result = awardNames.Reputation(100, true);
                expect(result.level).toBe(500); // Next after 250 is 500
            });
        });

        describe("Streak", () => {
            it("should return correct award for streak days", () => {
                // Based on awardVariants.Streak: [7, 30, 100, 200, 365, 500, 750, 1000]
                const result7 = awardNames.Streak(7);
                expect(result7.name).toBe("StreakDays30_Title"); // 7 < 30
                expect(result7.body).toBe("StreakDays_Body");
                expect(result7.bodyVariables).toEqual({ count: 7 });
                expect(result7.level).toBe(30);

                const result365 = awardNames.Streak(365);
                expect(result365.name).toBe("StreakDays500_Title"); // 365 < 500
                expect(result365.level).toBe(500);
            });
        });

        describe("RunRoutine", () => {
            it("should return correct award for routine runs", () => {
                const result1 = awardNames.RunRoutine(1);
                expect(result1.name).toBe("RunRoutine5_Title"); // 1 < 5
                expect(result1.body).toBe("RunRoutine_Body");
                expect(result1.level).toBe(5);

                const result10000 = awardNames.RunRoutine(10000);
                expect(result10000.name).toBe("RunRoutine10000_Title");
                expect(result10000.level).toBe(10000);
            });
        });

        describe("OrganizationCreate", () => {
            it("should return correct award for organization creation", () => {
                // Based on awardVariants.OrganizationCreate: [1, 2, 5, 10]
                const result1 = awardNames.OrganizationCreate(1);
                expect(result1.name).toBe("OrganizationCreate2_Title"); // 1 < 2
                expect(result1.body).toBe("OrganizationCreate_Body");
                expect(result1.level).toBe(2);

                const result10 = awardNames.OrganizationCreate(10);
                expect(result10.name).toBe("OrganizationCreate10_Title");
                expect(result10.level).toBe(10);
            });
        });

        describe("OrganizationJoin", () => {
            it("should return correct award for organization joining", () => {
                // Based on awardVariants.OrganizationJoin: [1, 5, 10, 25]
                const result5 = awardNames.OrganizationJoin(5);
                expect(result5.name).toBe("OrganizationJoin10_Title"); // 5 < 10
                expect(result5.body).toBe("OrganizationJoin_Body");
                expect(result5.level).toBe(10);

                const result25 = awardNames.OrganizationJoin(25);
                expect(result25.name).toBe("OrganizationJoin25_Title");
                expect(result25.level).toBe(25);
            });
        });

        describe("UserInvite", () => {
            it("should return correct award for user invitations", () => {
                // Based on awardVariants.UserInvite: [1, 5, 10, 25, 50, 100]
                const result1 = awardNames.UserInvite(1);
                expect(result1.name).toBe("UserInvite5_Title"); // 1 < 5
                expect(result1.body).toBe("UserInvite_Body");
                expect(result1.level).toBe(5);

                const result100 = awardNames.UserInvite(100);
                expect(result100.name).toBe("UserInvite100_Title");
                expect(result100.level).toBe(100);
            });
        });

        describe("edge cases", () => {
            it("should handle count beyond maximum tier", () => {
                // For ApiCreate, max is 50, test with 100
                const result = awardNames.ApiCreate(100);
                expect(result.name).toBe("ApiCreate50_Title");
                expect(result.level).toBe(50);
            });

            it("should handle count between tiers", () => {
                // For ApiCreate: [1, 5, 10, 25, 50], test count 7 (between 5 and 10)
                const result = awardNames.ApiCreate(7);
                expect(result.name).toBe("ApiCreate10_Title"); // 7 < 10
                expect(result.level).toBe(10);
            });

            it("should handle zero counts for AccountAnniversary", () => {
                const result = awardNames.AccountAnniversary(0);
                expect(result.nameVariables).toEqual({ count: 0 });
                expect(result.level).toBe(0);
            });

            it("should handle large counts for AccountAnniversary", () => {
                const result = awardNames.AccountAnniversary(100);
                expect(result.nameVariables).toEqual({ count: 100 });
                expect(result.level).toBe(100);
            });
        });

        describe("all award categories coverage", () => {
            it("should have functions for all award categories with mappings", () => {
                const awardNameKeys = Object.keys(awardNames);

                // Account for the fact that awardNames has all the categories from AwardCategory
                expect(awardNameKeys).toContain("RunProject");
                expect(awardNameKeys).toContain("RunRoutine");

                // All categories should exist
                Object.values(AwardCategory).forEach(category => {
                    expect(awardNameKeys).toContain(category);
                });
            });

            it("should return valid objects for all implemented award functions", () => {
                // Test all functions that actually exist in awardNames
                Object.keys(awardNames).forEach(categoryKey => {
                    const awardFunction = awardNames[categoryKey as keyof typeof awardNames];
                    expect(typeof awardFunction).toBe("function");

                    const result = awardFunction(1);
                    expect(result).toHaveProperty("name");
                    expect(result).toHaveProperty("body");
                    expect(result).toHaveProperty("level");
                    expect(typeof result.level).toBe("number");
                });
            });

            it("should handle AwardCategory enum values correctly", () => {
                // Test that AwardCategory values map correctly to awardNames functions
                Object.values(AwardCategory).forEach(category => {
                    // All categories should have direct mappings
                    expect(awardNames[category]).toBeDefined();
                    expect(typeof awardNames[category]).toBe("function");
                });
            });
        });

        describe("findNext functionality", () => {
            it("should work correctly for all award types with tiers", () => {
                const tieredCategories = Object.keys(awardVariants) as Array<keyof typeof awardVariants>;

                tieredCategories.forEach(category => {
                    const variants = awardVariants[category];
                    const firstTier = variants[0];

                    if (firstTier !== undefined) {
                        const current = awardNames[category](firstTier);
                        const next = awardNames[category](firstTier, true);

                        expect(next.level).toBeGreaterThan(current.level);
                    }
                });
            });

            it("should handle findNext at maximum tier", () => {
                const maxApiCreateTier = Math.max(...awardVariants.ApiCreate);
                const result = awardNames.ApiCreate(maxApiCreateTier, true);

                // Should return the last tier when already at maximum
                expect(result.level).toBe(maxApiCreateTier);
            });
        });
    });
});
