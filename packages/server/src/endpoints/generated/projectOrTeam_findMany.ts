export const projectOrTeam_findMany = {
  "fieldName": "projectOrTeams",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "projectOrTeams",
        "loc": {
          "start": 2024,
          "end": 2038
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 2039,
              "end": 2044
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 2047,
                "end": 2052
              }
            },
            "loc": {
              "start": 2046,
              "end": 2052
            }
          },
          "loc": {
            "start": 2039,
            "end": 2052
          }
        }
      ],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "edges",
              "loc": {
                "start": 2060,
                "end": 2065
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "cursor",
                    "loc": {
                      "start": 2076,
                      "end": 2082
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2076,
                    "end": 2082
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "node",
                    "loc": {
                      "start": 2091,
                      "end": 2095
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Project",
                            "loc": {
                              "start": 2117,
                              "end": 2124
                            }
                          },
                          "loc": {
                            "start": 2117,
                            "end": 2124
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Project_list",
                                "loc": {
                                  "start": 2146,
                                  "end": 2158
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 2143,
                                "end": 2158
                              }
                            }
                          ],
                          "loc": {
                            "start": 2125,
                            "end": 2172
                          }
                        },
                        "loc": {
                          "start": 2110,
                          "end": 2172
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Team",
                            "loc": {
                              "start": 2192,
                              "end": 2196
                            }
                          },
                          "loc": {
                            "start": 2192,
                            "end": 2196
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Team_list",
                                "loc": {
                                  "start": 2218,
                                  "end": 2227
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 2215,
                                "end": 2227
                              }
                            }
                          ],
                          "loc": {
                            "start": 2197,
                            "end": 2241
                          }
                        },
                        "loc": {
                          "start": 2185,
                          "end": 2241
                        }
                      }
                    ],
                    "loc": {
                      "start": 2096,
                      "end": 2251
                    }
                  },
                  "loc": {
                    "start": 2091,
                    "end": 2251
                  }
                }
              ],
              "loc": {
                "start": 2066,
                "end": 2257
              }
            },
            "loc": {
              "start": 2060,
              "end": 2257
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "pageInfo",
              "loc": {
                "start": 2262,
                "end": 2270
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "hasNextPage",
                    "loc": {
                      "start": 2281,
                      "end": 2292
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2281,
                    "end": 2292
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorProject",
                    "loc": {
                      "start": 2301,
                      "end": 2317
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2301,
                    "end": 2317
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorTeam",
                    "loc": {
                      "start": 2326,
                      "end": 2339
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2326,
                    "end": 2339
                  }
                }
              ],
              "loc": {
                "start": 2271,
                "end": 2345
              }
            },
            "loc": {
              "start": 2262,
              "end": 2345
            }
          }
        ],
        "loc": {
          "start": 2054,
          "end": 2349
        }
      },
      "loc": {
        "start": 2024,
        "end": 2349
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 32,
          "end": 34
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 32,
        "end": 34
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 35,
          "end": 45
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 35,
        "end": 45
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 46,
          "end": 56
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 46,
        "end": 56
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "color",
        "loc": {
          "start": 57,
          "end": 62
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 57,
        "end": 62
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "label",
        "loc": {
          "start": 63,
          "end": 68
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 63,
        "end": 68
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 69,
          "end": 74
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Team",
                "loc": {
                  "start": 88,
                  "end": 92
                }
              },
              "loc": {
                "start": 88,
                "end": 92
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Team_nav",
                    "loc": {
                      "start": 106,
                      "end": 114
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 103,
                    "end": 114
                  }
                }
              ],
              "loc": {
                "start": 93,
                "end": 120
              }
            },
            "loc": {
              "start": 81,
              "end": 120
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 132,
                  "end": 136
                }
              },
              "loc": {
                "start": 132,
                "end": 136
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 150,
                      "end": 158
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 147,
                    "end": 158
                  }
                }
              ],
              "loc": {
                "start": 137,
                "end": 164
              }
            },
            "loc": {
              "start": 125,
              "end": 164
            }
          }
        ],
        "loc": {
          "start": 75,
          "end": 166
        }
      },
      "loc": {
        "start": 69,
        "end": 166
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 167,
          "end": 170
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 177,
                "end": 186
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 177,
              "end": 186
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 191,
                "end": 200
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 191,
              "end": 200
            }
          }
        ],
        "loc": {
          "start": 171,
          "end": 202
        }
      },
      "loc": {
        "start": 167,
        "end": 202
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 240,
          "end": 242
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 240,
        "end": 242
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 243,
          "end": 253
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 243,
        "end": 253
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 254,
          "end": 264
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 254,
        "end": 264
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 265,
          "end": 274
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 265,
        "end": 274
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 275,
          "end": 286
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 275,
        "end": 286
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 287,
          "end": 293
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Label_list",
              "loc": {
                "start": 303,
                "end": 313
              }
            },
            "directives": [],
            "loc": {
              "start": 300,
              "end": 313
            }
          }
        ],
        "loc": {
          "start": 294,
          "end": 315
        }
      },
      "loc": {
        "start": 287,
        "end": 315
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 316,
          "end": 321
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Team",
                "loc": {
                  "start": 335,
                  "end": 339
                }
              },
              "loc": {
                "start": 335,
                "end": 339
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Team_nav",
                    "loc": {
                      "start": 353,
                      "end": 361
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 350,
                    "end": 361
                  }
                }
              ],
              "loc": {
                "start": 340,
                "end": 367
              }
            },
            "loc": {
              "start": 328,
              "end": 367
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 379,
                  "end": 383
                }
              },
              "loc": {
                "start": 379,
                "end": 383
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 397,
                      "end": 405
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 394,
                    "end": 405
                  }
                }
              ],
              "loc": {
                "start": 384,
                "end": 411
              }
            },
            "loc": {
              "start": 372,
              "end": 411
            }
          }
        ],
        "loc": {
          "start": 322,
          "end": 413
        }
      },
      "loc": {
        "start": 316,
        "end": 413
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 414,
          "end": 425
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 414,
        "end": 425
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 426,
          "end": 440
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 426,
        "end": 440
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 441,
          "end": 446
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 441,
        "end": 446
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 447,
          "end": 456
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 447,
        "end": 456
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 457,
          "end": 461
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 471,
                "end": 479
              }
            },
            "directives": [],
            "loc": {
              "start": 468,
              "end": 479
            }
          }
        ],
        "loc": {
          "start": 462,
          "end": 481
        }
      },
      "loc": {
        "start": 457,
        "end": 481
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 482,
          "end": 496
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 482,
        "end": 496
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 497,
          "end": 502
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 497,
        "end": 502
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 503,
          "end": 506
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 513,
                "end": 522
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 513,
              "end": 522
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 527,
                "end": 538
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 527,
              "end": 538
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 543,
                "end": 554
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 543,
              "end": 554
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 559,
                "end": 568
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 559,
              "end": 568
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 573,
                "end": 580
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 573,
              "end": 580
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 585,
                "end": 593
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 585,
              "end": 593
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 598,
                "end": 610
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 598,
              "end": 610
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 615,
                "end": 623
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 615,
              "end": 623
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 628,
                "end": 636
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 628,
              "end": 636
            }
          }
        ],
        "loc": {
          "start": 507,
          "end": 638
        }
      },
      "loc": {
        "start": 503,
        "end": 638
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 639,
          "end": 647
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 654,
                "end": 656
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 654,
              "end": 656
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 661,
                "end": 671
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 661,
              "end": 671
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 676,
                "end": 686
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 676,
              "end": 686
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoriesCount",
              "loc": {
                "start": 691,
                "end": 707
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 691,
              "end": 707
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 712,
                "end": 720
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 712,
              "end": 720
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 725,
                "end": 734
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 725,
              "end": 734
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 739,
                "end": 751
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 739,
              "end": 751
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "runProjectsCount",
              "loc": {
                "start": 756,
                "end": 772
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 756,
              "end": 772
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "simplicity",
              "loc": {
                "start": 777,
                "end": 787
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 777,
              "end": 787
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 792,
                "end": 804
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 792,
              "end": 804
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 809,
                "end": 821
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 809,
              "end": 821
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 826,
                "end": 838
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 849,
                      "end": 851
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 849,
                    "end": 851
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 860,
                      "end": 868
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 860,
                    "end": 868
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 877,
                      "end": 888
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 877,
                    "end": 888
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 897,
                      "end": 901
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 897,
                    "end": 901
                  }
                }
              ],
              "loc": {
                "start": 839,
                "end": 907
              }
            },
            "loc": {
              "start": 826,
              "end": 907
            }
          }
        ],
        "loc": {
          "start": 648,
          "end": 909
        }
      },
      "loc": {
        "start": 639,
        "end": 909
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 939,
          "end": 941
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 939,
        "end": 941
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 942,
          "end": 952
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 942,
        "end": 952
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tag",
        "loc": {
          "start": 953,
          "end": 956
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 953,
        "end": 956
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 957,
          "end": 966
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 957,
        "end": 966
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 967,
          "end": 979
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 986,
                "end": 988
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 986,
              "end": 988
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 993,
                "end": 1001
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 993,
              "end": 1001
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 1006,
                "end": 1017
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1006,
              "end": 1017
            }
          }
        ],
        "loc": {
          "start": 980,
          "end": 1019
        }
      },
      "loc": {
        "start": 967,
        "end": 1019
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1020,
          "end": 1023
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isOwn",
              "loc": {
                "start": 1030,
                "end": 1035
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1030,
              "end": 1035
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 1040,
                "end": 1052
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1040,
              "end": 1052
            }
          }
        ],
        "loc": {
          "start": 1024,
          "end": 1054
        }
      },
      "loc": {
        "start": 1020,
        "end": 1054
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1086,
          "end": 1088
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1086,
        "end": 1088
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 1089,
          "end": 1100
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1089,
        "end": 1100
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 1101,
          "end": 1107
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1101,
        "end": 1107
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1108,
          "end": 1118
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1108,
        "end": 1118
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 1119,
          "end": 1129
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1119,
        "end": 1129
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isOpenToNewMembers",
        "loc": {
          "start": 1130,
          "end": 1148
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1130,
        "end": 1148
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 1149,
          "end": 1158
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1149,
        "end": 1158
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "commentsCount",
        "loc": {
          "start": 1159,
          "end": 1172
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1159,
        "end": 1172
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "membersCount",
        "loc": {
          "start": 1173,
          "end": 1185
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1173,
        "end": 1185
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 1186,
          "end": 1198
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1186,
        "end": 1198
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reportsCount",
        "loc": {
          "start": 1199,
          "end": 1211
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1199,
        "end": 1211
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 1212,
          "end": 1221
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1212,
        "end": 1221
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 1222,
          "end": 1226
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 1236,
                "end": 1244
              }
            },
            "directives": [],
            "loc": {
              "start": 1233,
              "end": 1244
            }
          }
        ],
        "loc": {
          "start": 1227,
          "end": 1246
        }
      },
      "loc": {
        "start": 1222,
        "end": 1246
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 1247,
          "end": 1259
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1266,
                "end": 1268
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1266,
              "end": 1268
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 1273,
                "end": 1281
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1273,
              "end": 1281
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bio",
              "loc": {
                "start": 1286,
                "end": 1289
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1286,
              "end": 1289
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 1294,
                "end": 1298
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1294,
              "end": 1298
            }
          }
        ],
        "loc": {
          "start": 1260,
          "end": 1300
        }
      },
      "loc": {
        "start": 1247,
        "end": 1300
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1301,
          "end": 1304
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canAddMembers",
              "loc": {
                "start": 1311,
                "end": 1324
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1311,
              "end": 1324
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 1329,
                "end": 1338
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1329,
              "end": 1338
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 1343,
                "end": 1354
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1343,
              "end": 1354
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 1359,
                "end": 1368
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1359,
              "end": 1368
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 1373,
                "end": 1382
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1373,
              "end": 1382
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 1387,
                "end": 1394
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1387,
              "end": 1394
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 1399,
                "end": 1411
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1399,
              "end": 1411
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 1416,
                "end": 1424
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1416,
              "end": 1424
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 1429,
                "end": 1443
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1454,
                      "end": 1456
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1454,
                    "end": 1456
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1465,
                      "end": 1475
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1465,
                    "end": 1475
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1484,
                      "end": 1494
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1484,
                    "end": 1494
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 1503,
                      "end": 1510
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1503,
                    "end": 1510
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 1519,
                      "end": 1530
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1519,
                    "end": 1530
                  }
                }
              ],
              "loc": {
                "start": 1444,
                "end": 1536
              }
            },
            "loc": {
              "start": 1429,
              "end": 1536
            }
          }
        ],
        "loc": {
          "start": 1305,
          "end": 1538
        }
      },
      "loc": {
        "start": 1301,
        "end": 1538
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1569,
          "end": 1571
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1569,
        "end": 1571
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 1572,
          "end": 1583
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1572,
        "end": 1583
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 1584,
          "end": 1590
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1584,
        "end": 1590
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 1591,
          "end": 1603
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1591,
        "end": 1603
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1604,
          "end": 1607
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canAddMembers",
              "loc": {
                "start": 1614,
                "end": 1627
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1614,
              "end": 1627
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 1632,
                "end": 1641
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1632,
              "end": 1641
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 1646,
                "end": 1657
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1646,
              "end": 1657
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 1662,
                "end": 1671
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1662,
              "end": 1671
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 1676,
                "end": 1685
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1676,
              "end": 1685
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 1690,
                "end": 1697
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1690,
              "end": 1697
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 1702,
                "end": 1714
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1702,
              "end": 1714
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 1719,
                "end": 1727
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1719,
              "end": 1727
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 1732,
                "end": 1746
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1757,
                      "end": 1759
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1757,
                    "end": 1759
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1768,
                      "end": 1778
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1768,
                    "end": 1778
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1787,
                      "end": 1797
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1787,
                    "end": 1797
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 1806,
                      "end": 1813
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1806,
                    "end": 1813
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 1822,
                      "end": 1833
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1822,
                    "end": 1833
                  }
                }
              ],
              "loc": {
                "start": 1747,
                "end": 1839
              }
            },
            "loc": {
              "start": 1732,
              "end": 1839
            }
          }
        ],
        "loc": {
          "start": 1608,
          "end": 1841
        }
      },
      "loc": {
        "start": 1604,
        "end": 1841
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1872,
          "end": 1874
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1872,
        "end": 1874
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1875,
          "end": 1885
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1875,
        "end": 1885
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 1886,
          "end": 1896
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1886,
        "end": 1896
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 1897,
          "end": 1908
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1897,
        "end": 1908
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 1909,
          "end": 1915
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1909,
        "end": 1915
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 1916,
          "end": 1921
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1916,
        "end": 1921
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBotDepictingPerson",
        "loc": {
          "start": 1922,
          "end": 1942
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1922,
        "end": 1942
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 1943,
          "end": 1947
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1943,
        "end": 1947
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 1948,
          "end": 1960
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1948,
        "end": 1960
      }
    }
  ],
  "returnType": null,
  "parentType": null,
  "schema": null,
  "fragments": {
    "Label_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Label_list",
        "loc": {
          "start": 10,
          "end": 20
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Label",
          "loc": {
            "start": 24,
            "end": 29
          }
        },
        "loc": {
          "start": 24,
          "end": 29
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 32,
                "end": 34
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 32,
              "end": 34
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 35,
                "end": 45
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 35,
              "end": 45
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 46,
                "end": 56
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 46,
              "end": 56
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "color",
              "loc": {
                "start": 57,
                "end": 62
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 57,
              "end": 62
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "label",
              "loc": {
                "start": 63,
                "end": 68
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 63,
              "end": 68
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 69,
                "end": 74
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Team",
                      "loc": {
                        "start": 88,
                        "end": 92
                      }
                    },
                    "loc": {
                      "start": 88,
                      "end": 92
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Team_nav",
                          "loc": {
                            "start": 106,
                            "end": 114
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 103,
                          "end": 114
                        }
                      }
                    ],
                    "loc": {
                      "start": 93,
                      "end": 120
                    }
                  },
                  "loc": {
                    "start": 81,
                    "end": 120
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 132,
                        "end": 136
                      }
                    },
                    "loc": {
                      "start": 132,
                      "end": 136
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 150,
                            "end": 158
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 147,
                          "end": 158
                        }
                      }
                    ],
                    "loc": {
                      "start": 137,
                      "end": 164
                    }
                  },
                  "loc": {
                    "start": 125,
                    "end": 164
                  }
                }
              ],
              "loc": {
                "start": 75,
                "end": 166
              }
            },
            "loc": {
              "start": 69,
              "end": 166
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 167,
                "end": 170
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 177,
                      "end": 186
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 177,
                    "end": 186
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 191,
                      "end": 200
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 191,
                    "end": 200
                  }
                }
              ],
              "loc": {
                "start": 171,
                "end": 202
              }
            },
            "loc": {
              "start": 167,
              "end": 202
            }
          }
        ],
        "loc": {
          "start": 30,
          "end": 204
        }
      },
      "loc": {
        "start": 1,
        "end": 204
      }
    },
    "Project_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Project_list",
        "loc": {
          "start": 214,
          "end": 226
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Project",
          "loc": {
            "start": 230,
            "end": 237
          }
        },
        "loc": {
          "start": 230,
          "end": 237
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 240,
                "end": 242
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 240,
              "end": 242
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 243,
                "end": 253
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 243,
              "end": 253
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 254,
                "end": 264
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 254,
              "end": 264
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 265,
                "end": 274
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 265,
              "end": 274
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 275,
                "end": 286
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 275,
              "end": 286
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 287,
                "end": 293
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 303,
                      "end": 313
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 300,
                    "end": 313
                  }
                }
              ],
              "loc": {
                "start": 294,
                "end": 315
              }
            },
            "loc": {
              "start": 287,
              "end": 315
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 316,
                "end": 321
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Team",
                      "loc": {
                        "start": 335,
                        "end": 339
                      }
                    },
                    "loc": {
                      "start": 335,
                      "end": 339
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Team_nav",
                          "loc": {
                            "start": 353,
                            "end": 361
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 350,
                          "end": 361
                        }
                      }
                    ],
                    "loc": {
                      "start": 340,
                      "end": 367
                    }
                  },
                  "loc": {
                    "start": 328,
                    "end": 367
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 379,
                        "end": 383
                      }
                    },
                    "loc": {
                      "start": 379,
                      "end": 383
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 397,
                            "end": 405
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 394,
                          "end": 405
                        }
                      }
                    ],
                    "loc": {
                      "start": 384,
                      "end": 411
                    }
                  },
                  "loc": {
                    "start": 372,
                    "end": 411
                  }
                }
              ],
              "loc": {
                "start": 322,
                "end": 413
              }
            },
            "loc": {
              "start": 316,
              "end": 413
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 414,
                "end": 425
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 414,
              "end": 425
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 426,
                "end": 440
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 426,
              "end": 440
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 441,
                "end": 446
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 441,
              "end": 446
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 447,
                "end": 456
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 447,
              "end": 456
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 457,
                "end": 461
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 471,
                      "end": 479
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 468,
                    "end": 479
                  }
                }
              ],
              "loc": {
                "start": 462,
                "end": 481
              }
            },
            "loc": {
              "start": 457,
              "end": 481
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 482,
                "end": 496
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 482,
              "end": 496
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 497,
                "end": 502
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 497,
              "end": 502
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 503,
                "end": 506
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 513,
                      "end": 522
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 513,
                    "end": 522
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 527,
                      "end": 538
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 527,
                    "end": 538
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 543,
                      "end": 554
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 543,
                    "end": 554
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 559,
                      "end": 568
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 559,
                    "end": 568
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 573,
                      "end": 580
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 573,
                    "end": 580
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 585,
                      "end": 593
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 585,
                    "end": 593
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 598,
                      "end": 610
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 598,
                    "end": 610
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 615,
                      "end": 623
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 615,
                    "end": 623
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 628,
                      "end": 636
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 628,
                    "end": 636
                  }
                }
              ],
              "loc": {
                "start": 507,
                "end": 638
              }
            },
            "loc": {
              "start": 503,
              "end": 638
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 639,
                "end": 647
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 654,
                      "end": 656
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 654,
                    "end": 656
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 661,
                      "end": 671
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 661,
                    "end": 671
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 676,
                      "end": 686
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 676,
                    "end": 686
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoriesCount",
                    "loc": {
                      "start": 691,
                      "end": 707
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 691,
                    "end": 707
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 712,
                      "end": 720
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 712,
                    "end": 720
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 725,
                      "end": 734
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 725,
                    "end": 734
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 739,
                      "end": 751
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 739,
                    "end": 751
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "runProjectsCount",
                    "loc": {
                      "start": 756,
                      "end": 772
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 756,
                    "end": 772
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "simplicity",
                    "loc": {
                      "start": 777,
                      "end": 787
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 777,
                    "end": 787
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 792,
                      "end": 804
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 792,
                    "end": 804
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 809,
                      "end": 821
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 809,
                    "end": 821
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 826,
                      "end": 838
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 849,
                            "end": 851
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 849,
                          "end": 851
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 860,
                            "end": 868
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 860,
                          "end": 868
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 877,
                            "end": 888
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 877,
                          "end": 888
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 897,
                            "end": 901
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 897,
                          "end": 901
                        }
                      }
                    ],
                    "loc": {
                      "start": 839,
                      "end": 907
                    }
                  },
                  "loc": {
                    "start": 826,
                    "end": 907
                  }
                }
              ],
              "loc": {
                "start": 648,
                "end": 909
              }
            },
            "loc": {
              "start": 639,
              "end": 909
            }
          }
        ],
        "loc": {
          "start": 238,
          "end": 911
        }
      },
      "loc": {
        "start": 205,
        "end": 911
      }
    },
    "Tag_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Tag_list",
        "loc": {
          "start": 921,
          "end": 929
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Tag",
          "loc": {
            "start": 933,
            "end": 936
          }
        },
        "loc": {
          "start": 933,
          "end": 936
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 939,
                "end": 941
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 939,
              "end": 941
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 942,
                "end": 952
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 942,
              "end": 952
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tag",
              "loc": {
                "start": 953,
                "end": 956
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 953,
              "end": 956
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 957,
                "end": 966
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 957,
              "end": 966
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 967,
                "end": 979
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 986,
                      "end": 988
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 986,
                    "end": 988
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 993,
                      "end": 1001
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 993,
                    "end": 1001
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1006,
                      "end": 1017
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1006,
                    "end": 1017
                  }
                }
              ],
              "loc": {
                "start": 980,
                "end": 1019
              }
            },
            "loc": {
              "start": 967,
              "end": 1019
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1020,
                "end": 1023
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isOwn",
                    "loc": {
                      "start": 1030,
                      "end": 1035
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1030,
                    "end": 1035
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 1040,
                      "end": 1052
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1040,
                    "end": 1052
                  }
                }
              ],
              "loc": {
                "start": 1024,
                "end": 1054
              }
            },
            "loc": {
              "start": 1020,
              "end": 1054
            }
          }
        ],
        "loc": {
          "start": 937,
          "end": 1056
        }
      },
      "loc": {
        "start": 912,
        "end": 1056
      }
    },
    "Team_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Team_list",
        "loc": {
          "start": 1066,
          "end": 1075
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Team",
          "loc": {
            "start": 1079,
            "end": 1083
          }
        },
        "loc": {
          "start": 1079,
          "end": 1083
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1086,
                "end": 1088
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1086,
              "end": 1088
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 1089,
                "end": 1100
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1089,
              "end": 1100
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 1101,
                "end": 1107
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1101,
              "end": 1107
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1108,
                "end": 1118
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1108,
              "end": 1118
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1119,
                "end": 1129
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1119,
              "end": 1129
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isOpenToNewMembers",
              "loc": {
                "start": 1130,
                "end": 1148
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1130,
              "end": 1148
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1149,
                "end": 1158
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1149,
              "end": 1158
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 1159,
                "end": 1172
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1159,
              "end": 1172
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "membersCount",
              "loc": {
                "start": 1173,
                "end": 1185
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1173,
              "end": 1185
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 1186,
                "end": 1198
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1186,
              "end": 1198
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 1199,
                "end": 1211
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1199,
              "end": 1211
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 1212,
                "end": 1221
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1212,
              "end": 1221
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 1222,
                "end": 1226
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 1236,
                      "end": 1244
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1233,
                    "end": 1244
                  }
                }
              ],
              "loc": {
                "start": 1227,
                "end": 1246
              }
            },
            "loc": {
              "start": 1222,
              "end": 1246
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 1247,
                "end": 1259
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1266,
                      "end": 1268
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1266,
                    "end": 1268
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 1273,
                      "end": 1281
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1273,
                    "end": 1281
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bio",
                    "loc": {
                      "start": 1286,
                      "end": 1289
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1286,
                    "end": 1289
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 1294,
                      "end": 1298
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1294,
                    "end": 1298
                  }
                }
              ],
              "loc": {
                "start": 1260,
                "end": 1300
              }
            },
            "loc": {
              "start": 1247,
              "end": 1300
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1301,
                "end": 1304
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canAddMembers",
                    "loc": {
                      "start": 1311,
                      "end": 1324
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1311,
                    "end": 1324
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 1329,
                      "end": 1338
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1329,
                    "end": 1338
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 1343,
                      "end": 1354
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1343,
                    "end": 1354
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 1359,
                      "end": 1368
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1359,
                    "end": 1368
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1373,
                      "end": 1382
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1373,
                    "end": 1382
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1387,
                      "end": 1394
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1387,
                    "end": 1394
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 1399,
                      "end": 1411
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1399,
                    "end": 1411
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 1416,
                      "end": 1424
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1416,
                    "end": 1424
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 1429,
                      "end": 1443
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 1454,
                            "end": 1456
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1454,
                          "end": 1456
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 1465,
                            "end": 1475
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1465,
                          "end": 1475
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 1484,
                            "end": 1494
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1484,
                          "end": 1494
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 1503,
                            "end": 1510
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1503,
                          "end": 1510
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 1519,
                            "end": 1530
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1519,
                          "end": 1530
                        }
                      }
                    ],
                    "loc": {
                      "start": 1444,
                      "end": 1536
                    }
                  },
                  "loc": {
                    "start": 1429,
                    "end": 1536
                  }
                }
              ],
              "loc": {
                "start": 1305,
                "end": 1538
              }
            },
            "loc": {
              "start": 1301,
              "end": 1538
            }
          }
        ],
        "loc": {
          "start": 1084,
          "end": 1540
        }
      },
      "loc": {
        "start": 1057,
        "end": 1540
      }
    },
    "Team_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Team_nav",
        "loc": {
          "start": 1550,
          "end": 1558
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Team",
          "loc": {
            "start": 1562,
            "end": 1566
          }
        },
        "loc": {
          "start": 1562,
          "end": 1566
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1569,
                "end": 1571
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1569,
              "end": 1571
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 1572,
                "end": 1583
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1572,
              "end": 1583
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 1584,
                "end": 1590
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1584,
              "end": 1590
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 1591,
                "end": 1603
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1591,
              "end": 1603
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1604,
                "end": 1607
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canAddMembers",
                    "loc": {
                      "start": 1614,
                      "end": 1627
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1614,
                    "end": 1627
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 1632,
                      "end": 1641
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1632,
                    "end": 1641
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 1646,
                      "end": 1657
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1646,
                    "end": 1657
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 1662,
                      "end": 1671
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1662,
                    "end": 1671
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1676,
                      "end": 1685
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1676,
                    "end": 1685
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1690,
                      "end": 1697
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1690,
                    "end": 1697
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 1702,
                      "end": 1714
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1702,
                    "end": 1714
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 1719,
                      "end": 1727
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1719,
                    "end": 1727
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 1732,
                      "end": 1746
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 1757,
                            "end": 1759
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1757,
                          "end": 1759
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 1768,
                            "end": 1778
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1768,
                          "end": 1778
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 1787,
                            "end": 1797
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1787,
                          "end": 1797
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 1806,
                            "end": 1813
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1806,
                          "end": 1813
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 1822,
                            "end": 1833
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1822,
                          "end": 1833
                        }
                      }
                    ],
                    "loc": {
                      "start": 1747,
                      "end": 1839
                    }
                  },
                  "loc": {
                    "start": 1732,
                    "end": 1839
                  }
                }
              ],
              "loc": {
                "start": 1608,
                "end": 1841
              }
            },
            "loc": {
              "start": 1604,
              "end": 1841
            }
          }
        ],
        "loc": {
          "start": 1567,
          "end": 1843
        }
      },
      "loc": {
        "start": 1541,
        "end": 1843
      }
    },
    "User_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_nav",
        "loc": {
          "start": 1853,
          "end": 1861
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 1865,
            "end": 1869
          }
        },
        "loc": {
          "start": 1865,
          "end": 1869
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1872,
                "end": 1874
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1872,
              "end": 1874
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1875,
                "end": 1885
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1875,
              "end": 1885
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1886,
                "end": 1896
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1886,
              "end": 1896
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 1897,
                "end": 1908
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1897,
              "end": 1908
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 1909,
                "end": 1915
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1909,
              "end": 1915
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 1916,
                "end": 1921
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1916,
              "end": 1921
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBotDepictingPerson",
              "loc": {
                "start": 1922,
                "end": 1942
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1922,
              "end": 1942
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 1943,
                "end": 1947
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1943,
              "end": 1947
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 1948,
                "end": 1960
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1948,
              "end": 1960
            }
          }
        ],
        "loc": {
          "start": 1870,
          "end": 1962
        }
      },
      "loc": {
        "start": 1844,
        "end": 1962
      }
    }
  },
  "rootValue": {},
  "operation": {
    "kind": "OperationDefinition",
    "operation": "query",
    "name": {
      "kind": "Name",
      "value": "projectOrTeams",
      "loc": {
        "start": 1970,
        "end": 1984
      }
    },
    "variableDefinitions": [
      {
        "kind": "VariableDefinition",
        "variable": {
          "kind": "Variable",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 1986,
              "end": 1991
            }
          },
          "loc": {
            "start": 1985,
            "end": 1991
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "ProjectOrTeamSearchInput",
              "loc": {
                "start": 1993,
                "end": 2017
              }
            },
            "loc": {
              "start": 1993,
              "end": 2017
            }
          },
          "loc": {
            "start": 1993,
            "end": 2018
          }
        },
        "directives": [],
        "loc": {
          "start": 1985,
          "end": 2018
        }
      }
    ],
    "directives": [],
    "selectionSet": {
      "kind": "SelectionSet",
      "selections": [
        {
          "kind": "Field",
          "name": {
            "kind": "Name",
            "value": "projectOrTeams",
            "loc": {
              "start": 2024,
              "end": 2038
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 2039,
                  "end": 2044
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 2047,
                    "end": 2052
                  }
                },
                "loc": {
                  "start": 2046,
                  "end": 2052
                }
              },
              "loc": {
                "start": 2039,
                "end": 2052
              }
            }
          ],
          "directives": [],
          "selectionSet": {
            "kind": "SelectionSet",
            "selections": [
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "edges",
                  "loc": {
                    "start": 2060,
                    "end": 2065
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "cursor",
                        "loc": {
                          "start": 2076,
                          "end": 2082
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 2076,
                        "end": 2082
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "node",
                        "loc": {
                          "start": 2091,
                          "end": 2095
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "selectionSet": {
                        "kind": "SelectionSet",
                        "selections": [
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Project",
                                "loc": {
                                  "start": 2117,
                                  "end": 2124
                                }
                              },
                              "loc": {
                                "start": 2117,
                                "end": 2124
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Project_list",
                                    "loc": {
                                      "start": 2146,
                                      "end": 2158
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 2143,
                                    "end": 2158
                                  }
                                }
                              ],
                              "loc": {
                                "start": 2125,
                                "end": 2172
                              }
                            },
                            "loc": {
                              "start": 2110,
                              "end": 2172
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Team",
                                "loc": {
                                  "start": 2192,
                                  "end": 2196
                                }
                              },
                              "loc": {
                                "start": 2192,
                                "end": 2196
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Team_list",
                                    "loc": {
                                      "start": 2218,
                                      "end": 2227
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 2215,
                                    "end": 2227
                                  }
                                }
                              ],
                              "loc": {
                                "start": 2197,
                                "end": 2241
                              }
                            },
                            "loc": {
                              "start": 2185,
                              "end": 2241
                            }
                          }
                        ],
                        "loc": {
                          "start": 2096,
                          "end": 2251
                        }
                      },
                      "loc": {
                        "start": 2091,
                        "end": 2251
                      }
                    }
                  ],
                  "loc": {
                    "start": 2066,
                    "end": 2257
                  }
                },
                "loc": {
                  "start": 2060,
                  "end": 2257
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "pageInfo",
                  "loc": {
                    "start": 2262,
                    "end": 2270
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "hasNextPage",
                        "loc": {
                          "start": 2281,
                          "end": 2292
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 2281,
                        "end": 2292
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorProject",
                        "loc": {
                          "start": 2301,
                          "end": 2317
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 2301,
                        "end": 2317
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorTeam",
                        "loc": {
                          "start": 2326,
                          "end": 2339
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 2326,
                        "end": 2339
                      }
                    }
                  ],
                  "loc": {
                    "start": 2271,
                    "end": 2345
                  }
                },
                "loc": {
                  "start": 2262,
                  "end": 2345
                }
              }
            ],
            "loc": {
              "start": 2054,
              "end": 2349
            }
          },
          "loc": {
            "start": 2024,
            "end": 2349
          }
        }
      ],
      "loc": {
        "start": 2020,
        "end": 2351
      }
    },
    "loc": {
      "start": 1964,
      "end": 2351
    }
  },
  "variableValues": {},
  "path": {
    "key": "projectOrTeam_findMany"
  }
} as const;
