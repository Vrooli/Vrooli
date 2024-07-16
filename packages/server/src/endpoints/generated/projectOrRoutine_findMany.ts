export const projectOrRoutine_findMany = {
  "fieldName": "projectOrRoutines",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "projectOrRoutines",
        "loc": {
          "start": 2484,
          "end": 2501
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 2502,
              "end": 2507
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 2510,
                "end": 2515
              }
            },
            "loc": {
              "start": 2509,
              "end": 2515
            }
          },
          "loc": {
            "start": 2502,
            "end": 2515
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
                "start": 2523,
                "end": 2528
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
                      "start": 2539,
                      "end": 2545
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2539,
                    "end": 2545
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "node",
                    "loc": {
                      "start": 2554,
                      "end": 2558
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
                              "start": 2580,
                              "end": 2587
                            }
                          },
                          "loc": {
                            "start": 2580,
                            "end": 2587
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
                                  "start": 2609,
                                  "end": 2621
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 2606,
                                "end": 2621
                              }
                            }
                          ],
                          "loc": {
                            "start": 2588,
                            "end": 2635
                          }
                        },
                        "loc": {
                          "start": 2573,
                          "end": 2635
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Routine",
                            "loc": {
                              "start": 2655,
                              "end": 2662
                            }
                          },
                          "loc": {
                            "start": 2655,
                            "end": 2662
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
                                "value": "Routine_list",
                                "loc": {
                                  "start": 2684,
                                  "end": 2696
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 2681,
                                "end": 2696
                              }
                            }
                          ],
                          "loc": {
                            "start": 2663,
                            "end": 2710
                          }
                        },
                        "loc": {
                          "start": 2648,
                          "end": 2710
                        }
                      }
                    ],
                    "loc": {
                      "start": 2559,
                      "end": 2720
                    }
                  },
                  "loc": {
                    "start": 2554,
                    "end": 2720
                  }
                }
              ],
              "loc": {
                "start": 2529,
                "end": 2726
              }
            },
            "loc": {
              "start": 2523,
              "end": 2726
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "pageInfo",
              "loc": {
                "start": 2731,
                "end": 2739
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
                      "start": 2750,
                      "end": 2761
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2750,
                    "end": 2761
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorProject",
                    "loc": {
                      "start": 2770,
                      "end": 2786
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2770,
                    "end": 2786
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorRoutine",
                    "loc": {
                      "start": 2795,
                      "end": 2811
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2795,
                    "end": 2811
                  }
                }
              ],
              "loc": {
                "start": 2740,
                "end": 2817
              }
            },
            "loc": {
              "start": 2731,
              "end": 2817
            }
          }
        ],
        "loc": {
          "start": 2517,
          "end": 2821
        }
      },
      "loc": {
        "start": 2484,
        "end": 2821
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
          "start": 947,
          "end": 949
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 947,
        "end": 949
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 950,
          "end": 960
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 950,
        "end": 960
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 961,
          "end": 971
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 961,
        "end": 971
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isInternal",
        "loc": {
          "start": 972,
          "end": 982
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 972,
        "end": 982
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 983,
          "end": 992
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 983,
        "end": 992
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 993,
          "end": 1004
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 993,
        "end": 1004
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 1005,
          "end": 1011
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
                "start": 1021,
                "end": 1031
              }
            },
            "directives": [],
            "loc": {
              "start": 1018,
              "end": 1031
            }
          }
        ],
        "loc": {
          "start": 1012,
          "end": 1033
        }
      },
      "loc": {
        "start": 1005,
        "end": 1033
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 1034,
          "end": 1039
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
                  "start": 1053,
                  "end": 1057
                }
              },
              "loc": {
                "start": 1053,
                "end": 1057
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
                      "start": 1071,
                      "end": 1079
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1068,
                    "end": 1079
                  }
                }
              ],
              "loc": {
                "start": 1058,
                "end": 1085
              }
            },
            "loc": {
              "start": 1046,
              "end": 1085
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
                  "start": 1097,
                  "end": 1101
                }
              },
              "loc": {
                "start": 1097,
                "end": 1101
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
                      "start": 1115,
                      "end": 1123
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1112,
                    "end": 1123
                  }
                }
              ],
              "loc": {
                "start": 1102,
                "end": 1129
              }
            },
            "loc": {
              "start": 1090,
              "end": 1129
            }
          }
        ],
        "loc": {
          "start": 1040,
          "end": 1131
        }
      },
      "loc": {
        "start": 1034,
        "end": 1131
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 1132,
          "end": 1143
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1132,
        "end": 1143
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 1144,
          "end": 1158
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1144,
        "end": 1158
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 1159,
          "end": 1164
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1159,
        "end": 1164
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 1165,
          "end": 1174
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1165,
        "end": 1174
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 1175,
          "end": 1179
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
                "start": 1189,
                "end": 1197
              }
            },
            "directives": [],
            "loc": {
              "start": 1186,
              "end": 1197
            }
          }
        ],
        "loc": {
          "start": 1180,
          "end": 1199
        }
      },
      "loc": {
        "start": 1175,
        "end": 1199
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 1200,
          "end": 1214
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1200,
        "end": 1214
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 1215,
          "end": 1220
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1215,
        "end": 1220
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1221,
          "end": 1224
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
              "value": "canComment",
              "loc": {
                "start": 1231,
                "end": 1241
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1231,
              "end": 1241
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 1246,
                "end": 1255
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1246,
              "end": 1255
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 1260,
                "end": 1271
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1260,
              "end": 1271
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 1276,
                "end": 1285
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1276,
              "end": 1285
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 1290,
                "end": 1297
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1290,
              "end": 1297
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 1302,
                "end": 1310
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1302,
              "end": 1310
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 1315,
                "end": 1327
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1315,
              "end": 1327
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 1332,
                "end": 1340
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1332,
              "end": 1340
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 1345,
                "end": 1353
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1345,
              "end": 1353
            }
          }
        ],
        "loc": {
          "start": 1225,
          "end": 1355
        }
      },
      "loc": {
        "start": 1221,
        "end": 1355
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 1356,
          "end": 1364
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
                "start": 1371,
                "end": 1373
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1371,
              "end": 1373
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1378,
                "end": 1388
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1378,
              "end": 1388
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1393,
                "end": 1403
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1393,
              "end": 1403
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedAt",
              "loc": {
                "start": 1408,
                "end": 1419
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1408,
              "end": 1419
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isAutomatable",
              "loc": {
                "start": 1424,
                "end": 1437
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1424,
              "end": 1437
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 1442,
                "end": 1452
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1442,
              "end": 1452
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isDeleted",
              "loc": {
                "start": 1457,
                "end": 1466
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1457,
              "end": 1466
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 1471,
                "end": 1479
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1471,
              "end": 1479
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1484,
                "end": 1493
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1484,
              "end": 1493
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "routineType",
              "loc": {
                "start": 1498,
                "end": 1509
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1498,
              "end": 1509
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "simplicity",
              "loc": {
                "start": 1514,
                "end": 1524
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1514,
              "end": 1524
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timesStarted",
              "loc": {
                "start": 1529,
                "end": 1541
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1529,
              "end": 1541
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timesCompleted",
              "loc": {
                "start": 1546,
                "end": 1560
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1546,
              "end": 1560
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 1565,
                "end": 1577
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1565,
              "end": 1577
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 1582,
                "end": 1594
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1582,
              "end": 1594
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 1599,
                "end": 1612
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1599,
              "end": 1612
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoryListingsCount",
              "loc": {
                "start": 1617,
                "end": 1639
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1617,
              "end": 1639
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 1644,
                "end": 1654
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1644,
              "end": 1654
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "inputsCount",
              "loc": {
                "start": 1659,
                "end": 1670
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1659,
              "end": 1670
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "nodesCount",
              "loc": {
                "start": 1675,
                "end": 1685
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1675,
              "end": 1685
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "nodeLinksCount",
              "loc": {
                "start": 1690,
                "end": 1704
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1690,
              "end": 1704
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "outputsCount",
              "loc": {
                "start": 1709,
                "end": 1721
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1709,
              "end": 1721
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 1726,
                "end": 1738
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1726,
              "end": 1738
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 1743,
                "end": 1755
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
                      "start": 1766,
                      "end": 1768
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1766,
                    "end": 1768
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 1777,
                      "end": 1785
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1777,
                    "end": 1785
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1794,
                      "end": 1805
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1794,
                    "end": 1805
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "instructions",
                    "loc": {
                      "start": 1814,
                      "end": 1826
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1814,
                    "end": 1826
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 1835,
                      "end": 1839
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1835,
                    "end": 1839
                  }
                }
              ],
              "loc": {
                "start": 1756,
                "end": 1845
              }
            },
            "loc": {
              "start": 1743,
              "end": 1845
            }
          }
        ],
        "loc": {
          "start": 1365,
          "end": 1847
        }
      },
      "loc": {
        "start": 1356,
        "end": 1847
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1877,
          "end": 1879
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1877,
        "end": 1879
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1880,
          "end": 1890
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1880,
        "end": 1890
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tag",
        "loc": {
          "start": 1891,
          "end": 1894
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1891,
        "end": 1894
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 1895,
          "end": 1904
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1895,
        "end": 1904
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 1905,
          "end": 1917
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
                "start": 1924,
                "end": 1926
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1924,
              "end": 1926
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 1931,
                "end": 1939
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1931,
              "end": 1939
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 1944,
                "end": 1955
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1944,
              "end": 1955
            }
          }
        ],
        "loc": {
          "start": 1918,
          "end": 1957
        }
      },
      "loc": {
        "start": 1905,
        "end": 1957
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1958,
          "end": 1961
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
                "start": 1968,
                "end": 1973
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1968,
              "end": 1973
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 1978,
                "end": 1990
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1978,
              "end": 1990
            }
          }
        ],
        "loc": {
          "start": 1962,
          "end": 1992
        }
      },
      "loc": {
        "start": 1958,
        "end": 1992
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 2023,
          "end": 2025
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2023,
        "end": 2025
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 2026,
          "end": 2037
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2026,
        "end": 2037
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 2038,
          "end": 2044
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2038,
        "end": 2044
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 2045,
          "end": 2057
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2045,
        "end": 2057
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 2058,
          "end": 2061
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
                "start": 2068,
                "end": 2081
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2068,
              "end": 2081
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 2086,
                "end": 2095
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2086,
              "end": 2095
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 2100,
                "end": 2111
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2100,
              "end": 2111
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 2116,
                "end": 2125
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2116,
              "end": 2125
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 2130,
                "end": 2139
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2130,
              "end": 2139
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 2144,
                "end": 2151
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2144,
              "end": 2151
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 2156,
                "end": 2168
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2156,
              "end": 2168
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 2173,
                "end": 2181
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2173,
              "end": 2181
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 2186,
                "end": 2200
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
                      "start": 2211,
                      "end": 2213
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2211,
                    "end": 2213
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 2222,
                      "end": 2232
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2222,
                    "end": 2232
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 2241,
                      "end": 2251
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2241,
                    "end": 2251
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 2260,
                      "end": 2267
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2260,
                    "end": 2267
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 2276,
                      "end": 2287
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2276,
                    "end": 2287
                  }
                }
              ],
              "loc": {
                "start": 2201,
                "end": 2293
              }
            },
            "loc": {
              "start": 2186,
              "end": 2293
            }
          }
        ],
        "loc": {
          "start": 2062,
          "end": 2295
        }
      },
      "loc": {
        "start": 2058,
        "end": 2295
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 2326,
          "end": 2328
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2326,
        "end": 2328
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 2329,
          "end": 2339
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2329,
        "end": 2339
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 2340,
          "end": 2350
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2340,
        "end": 2350
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 2351,
          "end": 2362
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2351,
        "end": 2362
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 2363,
          "end": 2369
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2363,
        "end": 2369
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 2370,
          "end": 2375
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2370,
        "end": 2375
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBotDepictingPerson",
        "loc": {
          "start": 2376,
          "end": 2396
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2376,
        "end": 2396
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 2397,
          "end": 2401
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2397,
        "end": 2401
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 2402,
          "end": 2414
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2402,
        "end": 2414
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
    "Routine_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Routine_list",
        "loc": {
          "start": 921,
          "end": 933
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Routine",
          "loc": {
            "start": 937,
            "end": 944
          }
        },
        "loc": {
          "start": 937,
          "end": 944
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
                "start": 947,
                "end": 949
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 947,
              "end": 949
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 950,
                "end": 960
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 950,
              "end": 960
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 961,
                "end": 971
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 961,
              "end": 971
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isInternal",
              "loc": {
                "start": 972,
                "end": 982
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 972,
              "end": 982
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 983,
                "end": 992
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 983,
              "end": 992
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 993,
                "end": 1004
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 993,
              "end": 1004
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 1005,
                "end": 1011
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
                      "start": 1021,
                      "end": 1031
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1018,
                    "end": 1031
                  }
                }
              ],
              "loc": {
                "start": 1012,
                "end": 1033
              }
            },
            "loc": {
              "start": 1005,
              "end": 1033
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 1034,
                "end": 1039
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
                        "start": 1053,
                        "end": 1057
                      }
                    },
                    "loc": {
                      "start": 1053,
                      "end": 1057
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
                            "start": 1071,
                            "end": 1079
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1068,
                          "end": 1079
                        }
                      }
                    ],
                    "loc": {
                      "start": 1058,
                      "end": 1085
                    }
                  },
                  "loc": {
                    "start": 1046,
                    "end": 1085
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
                        "start": 1097,
                        "end": 1101
                      }
                    },
                    "loc": {
                      "start": 1097,
                      "end": 1101
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
                            "start": 1115,
                            "end": 1123
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1112,
                          "end": 1123
                        }
                      }
                    ],
                    "loc": {
                      "start": 1102,
                      "end": 1129
                    }
                  },
                  "loc": {
                    "start": 1090,
                    "end": 1129
                  }
                }
              ],
              "loc": {
                "start": 1040,
                "end": 1131
              }
            },
            "loc": {
              "start": 1034,
              "end": 1131
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 1132,
                "end": 1143
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1132,
              "end": 1143
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 1144,
                "end": 1158
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1144,
              "end": 1158
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 1159,
                "end": 1164
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1159,
              "end": 1164
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 1165,
                "end": 1174
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1165,
              "end": 1174
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 1175,
                "end": 1179
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
                      "start": 1189,
                      "end": 1197
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1186,
                    "end": 1197
                  }
                }
              ],
              "loc": {
                "start": 1180,
                "end": 1199
              }
            },
            "loc": {
              "start": 1175,
              "end": 1199
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 1200,
                "end": 1214
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1200,
              "end": 1214
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 1215,
                "end": 1220
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1215,
              "end": 1220
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1221,
                "end": 1224
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
                    "value": "canComment",
                    "loc": {
                      "start": 1231,
                      "end": 1241
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1231,
                    "end": 1241
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 1246,
                      "end": 1255
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1246,
                    "end": 1255
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 1260,
                      "end": 1271
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1260,
                    "end": 1271
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1276,
                      "end": 1285
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1276,
                    "end": 1285
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1290,
                      "end": 1297
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1290,
                    "end": 1297
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 1302,
                      "end": 1310
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1302,
                    "end": 1310
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 1315,
                      "end": 1327
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1315,
                    "end": 1327
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 1332,
                      "end": 1340
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1332,
                    "end": 1340
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 1345,
                      "end": 1353
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1345,
                    "end": 1353
                  }
                }
              ],
              "loc": {
                "start": 1225,
                "end": 1355
              }
            },
            "loc": {
              "start": 1221,
              "end": 1355
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 1356,
                "end": 1364
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
                      "start": 1371,
                      "end": 1373
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1371,
                    "end": 1373
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1378,
                      "end": 1388
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1378,
                    "end": 1388
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1393,
                      "end": 1403
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1393,
                    "end": 1403
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedAt",
                    "loc": {
                      "start": 1408,
                      "end": 1419
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1408,
                    "end": 1419
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAutomatable",
                    "loc": {
                      "start": 1424,
                      "end": 1437
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1424,
                    "end": 1437
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 1442,
                      "end": 1452
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1442,
                    "end": 1452
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isDeleted",
                    "loc": {
                      "start": 1457,
                      "end": 1466
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1457,
                    "end": 1466
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 1471,
                      "end": 1479
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1471,
                    "end": 1479
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 1484,
                      "end": 1493
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1484,
                    "end": 1493
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "routineType",
                    "loc": {
                      "start": 1498,
                      "end": 1509
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1498,
                    "end": 1509
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "simplicity",
                    "loc": {
                      "start": 1514,
                      "end": 1524
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1514,
                    "end": 1524
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timesStarted",
                    "loc": {
                      "start": 1529,
                      "end": 1541
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1529,
                    "end": 1541
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timesCompleted",
                    "loc": {
                      "start": 1546,
                      "end": 1560
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1546,
                    "end": 1560
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 1565,
                      "end": 1577
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1565,
                    "end": 1577
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 1582,
                      "end": 1594
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1582,
                    "end": 1594
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 1599,
                      "end": 1612
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1599,
                    "end": 1612
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 1617,
                      "end": 1639
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1617,
                    "end": 1639
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 1644,
                      "end": 1654
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1644,
                    "end": 1654
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "inputsCount",
                    "loc": {
                      "start": 1659,
                      "end": 1670
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1659,
                    "end": 1670
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "nodesCount",
                    "loc": {
                      "start": 1675,
                      "end": 1685
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1675,
                    "end": 1685
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "nodeLinksCount",
                    "loc": {
                      "start": 1690,
                      "end": 1704
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1690,
                    "end": 1704
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "outputsCount",
                    "loc": {
                      "start": 1709,
                      "end": 1721
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1709,
                    "end": 1721
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 1726,
                      "end": 1738
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1726,
                    "end": 1738
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 1743,
                      "end": 1755
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
                            "start": 1766,
                            "end": 1768
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1766,
                          "end": 1768
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 1777,
                            "end": 1785
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1777,
                          "end": 1785
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 1794,
                            "end": 1805
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1794,
                          "end": 1805
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "instructions",
                          "loc": {
                            "start": 1814,
                            "end": 1826
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1814,
                          "end": 1826
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 1835,
                            "end": 1839
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1835,
                          "end": 1839
                        }
                      }
                    ],
                    "loc": {
                      "start": 1756,
                      "end": 1845
                    }
                  },
                  "loc": {
                    "start": 1743,
                    "end": 1845
                  }
                }
              ],
              "loc": {
                "start": 1365,
                "end": 1847
              }
            },
            "loc": {
              "start": 1356,
              "end": 1847
            }
          }
        ],
        "loc": {
          "start": 945,
          "end": 1849
        }
      },
      "loc": {
        "start": 912,
        "end": 1849
      }
    },
    "Tag_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Tag_list",
        "loc": {
          "start": 1859,
          "end": 1867
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Tag",
          "loc": {
            "start": 1871,
            "end": 1874
          }
        },
        "loc": {
          "start": 1871,
          "end": 1874
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
                "start": 1877,
                "end": 1879
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1877,
              "end": 1879
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1880,
                "end": 1890
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1880,
              "end": 1890
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tag",
              "loc": {
                "start": 1891,
                "end": 1894
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1891,
              "end": 1894
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 1895,
                "end": 1904
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1895,
              "end": 1904
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 1905,
                "end": 1917
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
                      "start": 1924,
                      "end": 1926
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1924,
                    "end": 1926
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 1931,
                      "end": 1939
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1931,
                    "end": 1939
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1944,
                      "end": 1955
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1944,
                    "end": 1955
                  }
                }
              ],
              "loc": {
                "start": 1918,
                "end": 1957
              }
            },
            "loc": {
              "start": 1905,
              "end": 1957
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1958,
                "end": 1961
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
                      "start": 1968,
                      "end": 1973
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1968,
                    "end": 1973
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 1978,
                      "end": 1990
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1978,
                    "end": 1990
                  }
                }
              ],
              "loc": {
                "start": 1962,
                "end": 1992
              }
            },
            "loc": {
              "start": 1958,
              "end": 1992
            }
          }
        ],
        "loc": {
          "start": 1875,
          "end": 1994
        }
      },
      "loc": {
        "start": 1850,
        "end": 1994
      }
    },
    "Team_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Team_nav",
        "loc": {
          "start": 2004,
          "end": 2012
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Team",
          "loc": {
            "start": 2016,
            "end": 2020
          }
        },
        "loc": {
          "start": 2016,
          "end": 2020
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
                "start": 2023,
                "end": 2025
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2023,
              "end": 2025
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 2026,
                "end": 2037
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2026,
              "end": 2037
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 2038,
                "end": 2044
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2038,
              "end": 2044
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 2045,
                "end": 2057
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2045,
              "end": 2057
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2058,
                "end": 2061
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
                      "start": 2068,
                      "end": 2081
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2068,
                    "end": 2081
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 2086,
                      "end": 2095
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2086,
                    "end": 2095
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 2100,
                      "end": 2111
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2100,
                    "end": 2111
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 2116,
                      "end": 2125
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2116,
                    "end": 2125
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 2130,
                      "end": 2139
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2130,
                    "end": 2139
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 2144,
                      "end": 2151
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2144,
                    "end": 2151
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 2156,
                      "end": 2168
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2156,
                    "end": 2168
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 2173,
                      "end": 2181
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2173,
                    "end": 2181
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 2186,
                      "end": 2200
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
                            "start": 2211,
                            "end": 2213
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2211,
                          "end": 2213
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 2222,
                            "end": 2232
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2222,
                          "end": 2232
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 2241,
                            "end": 2251
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2241,
                          "end": 2251
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 2260,
                            "end": 2267
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2260,
                          "end": 2267
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 2276,
                            "end": 2287
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2276,
                          "end": 2287
                        }
                      }
                    ],
                    "loc": {
                      "start": 2201,
                      "end": 2293
                    }
                  },
                  "loc": {
                    "start": 2186,
                    "end": 2293
                  }
                }
              ],
              "loc": {
                "start": 2062,
                "end": 2295
              }
            },
            "loc": {
              "start": 2058,
              "end": 2295
            }
          }
        ],
        "loc": {
          "start": 2021,
          "end": 2297
        }
      },
      "loc": {
        "start": 1995,
        "end": 2297
      }
    },
    "User_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_nav",
        "loc": {
          "start": 2307,
          "end": 2315
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 2319,
            "end": 2323
          }
        },
        "loc": {
          "start": 2319,
          "end": 2323
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
                "start": 2326,
                "end": 2328
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2326,
              "end": 2328
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 2329,
                "end": 2339
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2329,
              "end": 2339
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 2340,
                "end": 2350
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2340,
              "end": 2350
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 2351,
                "end": 2362
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2351,
              "end": 2362
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 2363,
                "end": 2369
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2363,
              "end": 2369
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 2370,
                "end": 2375
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2370,
              "end": 2375
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBotDepictingPerson",
              "loc": {
                "start": 2376,
                "end": 2396
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2376,
              "end": 2396
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 2397,
                "end": 2401
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2397,
              "end": 2401
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 2402,
                "end": 2414
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2402,
              "end": 2414
            }
          }
        ],
        "loc": {
          "start": 2324,
          "end": 2416
        }
      },
      "loc": {
        "start": 2298,
        "end": 2416
      }
    }
  },
  "rootValue": {},
  "operation": {
    "kind": "OperationDefinition",
    "operation": "query",
    "name": {
      "kind": "Name",
      "value": "projectOrRoutines",
      "loc": {
        "start": 2424,
        "end": 2441
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
              "start": 2443,
              "end": 2448
            }
          },
          "loc": {
            "start": 2442,
            "end": 2448
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "ProjectOrRoutineSearchInput",
              "loc": {
                "start": 2450,
                "end": 2477
              }
            },
            "loc": {
              "start": 2450,
              "end": 2477
            }
          },
          "loc": {
            "start": 2450,
            "end": 2478
          }
        },
        "directives": [],
        "loc": {
          "start": 2442,
          "end": 2478
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
            "value": "projectOrRoutines",
            "loc": {
              "start": 2484,
              "end": 2501
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 2502,
                  "end": 2507
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 2510,
                    "end": 2515
                  }
                },
                "loc": {
                  "start": 2509,
                  "end": 2515
                }
              },
              "loc": {
                "start": 2502,
                "end": 2515
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
                    "start": 2523,
                    "end": 2528
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
                          "start": 2539,
                          "end": 2545
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 2539,
                        "end": 2545
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "node",
                        "loc": {
                          "start": 2554,
                          "end": 2558
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
                                  "start": 2580,
                                  "end": 2587
                                }
                              },
                              "loc": {
                                "start": 2580,
                                "end": 2587
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
                                      "start": 2609,
                                      "end": 2621
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 2606,
                                    "end": 2621
                                  }
                                }
                              ],
                              "loc": {
                                "start": 2588,
                                "end": 2635
                              }
                            },
                            "loc": {
                              "start": 2573,
                              "end": 2635
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Routine",
                                "loc": {
                                  "start": 2655,
                                  "end": 2662
                                }
                              },
                              "loc": {
                                "start": 2655,
                                "end": 2662
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
                                    "value": "Routine_list",
                                    "loc": {
                                      "start": 2684,
                                      "end": 2696
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 2681,
                                    "end": 2696
                                  }
                                }
                              ],
                              "loc": {
                                "start": 2663,
                                "end": 2710
                              }
                            },
                            "loc": {
                              "start": 2648,
                              "end": 2710
                            }
                          }
                        ],
                        "loc": {
                          "start": 2559,
                          "end": 2720
                        }
                      },
                      "loc": {
                        "start": 2554,
                        "end": 2720
                      }
                    }
                  ],
                  "loc": {
                    "start": 2529,
                    "end": 2726
                  }
                },
                "loc": {
                  "start": 2523,
                  "end": 2726
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "pageInfo",
                  "loc": {
                    "start": 2731,
                    "end": 2739
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
                          "start": 2750,
                          "end": 2761
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 2750,
                        "end": 2761
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorProject",
                        "loc": {
                          "start": 2770,
                          "end": 2786
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 2770,
                        "end": 2786
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorRoutine",
                        "loc": {
                          "start": 2795,
                          "end": 2811
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 2795,
                        "end": 2811
                      }
                    }
                  ],
                  "loc": {
                    "start": 2740,
                    "end": 2817
                  }
                },
                "loc": {
                  "start": 2731,
                  "end": 2817
                }
              }
            ],
            "loc": {
              "start": 2517,
              "end": 2821
            }
          },
          "loc": {
            "start": 2484,
            "end": 2821
          }
        }
      ],
      "loc": {
        "start": 2480,
        "end": 2823
      }
    },
    "loc": {
      "start": 2418,
      "end": 2823
    }
  },
  "variableValues": {},
  "path": {
    "key": "projectOrRoutine_findMany"
  }
} as const;
