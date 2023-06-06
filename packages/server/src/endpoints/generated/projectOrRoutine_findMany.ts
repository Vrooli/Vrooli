export const projectOrRoutine_findMany = {
  "fieldName": "projectOrRoutines",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "projectOrRoutines",
        "loc": {
          "start": 2842,
          "end": 2859
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 2860,
              "end": 2865
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 2868,
                "end": 2873
              }
            },
            "loc": {
              "start": 2867,
              "end": 2873
            }
          },
          "loc": {
            "start": 2860,
            "end": 2873
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
                "start": 2881,
                "end": 2886
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
                      "start": 2897,
                      "end": 2903
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2897,
                    "end": 2903
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "node",
                    "loc": {
                      "start": 2912,
                      "end": 2916
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
                              "start": 2938,
                              "end": 2945
                            }
                          },
                          "loc": {
                            "start": 2938,
                            "end": 2945
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
                                  "start": 2967,
                                  "end": 2979
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 2964,
                                "end": 2979
                              }
                            }
                          ],
                          "loc": {
                            "start": 2946,
                            "end": 2993
                          }
                        },
                        "loc": {
                          "start": 2931,
                          "end": 2993
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
                              "start": 3013,
                              "end": 3020
                            }
                          },
                          "loc": {
                            "start": 3013,
                            "end": 3020
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
                                  "start": 3042,
                                  "end": 3054
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 3039,
                                "end": 3054
                              }
                            }
                          ],
                          "loc": {
                            "start": 3021,
                            "end": 3068
                          }
                        },
                        "loc": {
                          "start": 3006,
                          "end": 3068
                        }
                      }
                    ],
                    "loc": {
                      "start": 2917,
                      "end": 3078
                    }
                  },
                  "loc": {
                    "start": 2912,
                    "end": 3078
                  }
                }
              ],
              "loc": {
                "start": 2887,
                "end": 3084
              }
            },
            "loc": {
              "start": 2881,
              "end": 3084
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "pageInfo",
              "loc": {
                "start": 3089,
                "end": 3097
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
                      "start": 3108,
                      "end": 3119
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3108,
                    "end": 3119
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorProject",
                    "loc": {
                      "start": 3128,
                      "end": 3144
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3128,
                    "end": 3144
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorRoutine",
                    "loc": {
                      "start": 3153,
                      "end": 3169
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3153,
                    "end": 3169
                  }
                }
              ],
              "loc": {
                "start": 3098,
                "end": 3175
              }
            },
            "loc": {
              "start": 3089,
              "end": 3175
            }
          }
        ],
        "loc": {
          "start": 2875,
          "end": 3179
        }
      },
      "loc": {
        "start": 2842,
        "end": 3179
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "apisCount",
        "loc": {
          "start": 32,
          "end": 41
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 32,
        "end": 41
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "focusModesCount",
        "loc": {
          "start": 42,
          "end": 57
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 42,
        "end": 57
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 58,
          "end": 69
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 58,
        "end": 69
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "meetingsCount",
        "loc": {
          "start": 70,
          "end": 83
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 70,
        "end": 83
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "notesCount",
        "loc": {
          "start": 84,
          "end": 94
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 84,
        "end": 94
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "projectsCount",
        "loc": {
          "start": 95,
          "end": 108
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 95,
        "end": 108
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "routinesCount",
        "loc": {
          "start": 109,
          "end": 122
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 109,
        "end": 122
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "schedulesCount",
        "loc": {
          "start": 123,
          "end": 137
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 123,
        "end": 137
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "smartContractsCount",
        "loc": {
          "start": 138,
          "end": 157
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 138,
        "end": 157
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "standardsCount",
        "loc": {
          "start": 158,
          "end": 172
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 158,
        "end": 172
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 173,
          "end": 175
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 173,
        "end": 175
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 176,
          "end": 186
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 176,
        "end": 186
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 187,
          "end": 197
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 187,
        "end": 197
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "color",
        "loc": {
          "start": 198,
          "end": 203
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 198,
        "end": 203
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "label",
        "loc": {
          "start": 204,
          "end": 209
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 204,
        "end": 209
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 210,
          "end": 215
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
                "value": "Organization",
                "loc": {
                  "start": 229,
                  "end": 241
                }
              },
              "loc": {
                "start": 229,
                "end": 241
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
                    "value": "Organization_nav",
                    "loc": {
                      "start": 255,
                      "end": 271
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 252,
                    "end": 271
                  }
                }
              ],
              "loc": {
                "start": 242,
                "end": 277
              }
            },
            "loc": {
              "start": 222,
              "end": 277
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
                  "start": 289,
                  "end": 293
                }
              },
              "loc": {
                "start": 289,
                "end": 293
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
                      "start": 307,
                      "end": 315
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 304,
                    "end": 315
                  }
                }
              ],
              "loc": {
                "start": 294,
                "end": 321
              }
            },
            "loc": {
              "start": 282,
              "end": 321
            }
          }
        ],
        "loc": {
          "start": 216,
          "end": 323
        }
      },
      "loc": {
        "start": 210,
        "end": 323
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 324,
          "end": 327
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
                "start": 334,
                "end": 343
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 334,
              "end": 343
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 348,
                "end": 357
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 348,
              "end": 357
            }
          }
        ],
        "loc": {
          "start": 328,
          "end": 359
        }
      },
      "loc": {
        "start": 324,
        "end": 359
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 393,
          "end": 395
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 393,
        "end": 395
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 396,
          "end": 406
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 396,
        "end": 406
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 407,
          "end": 417
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 407,
        "end": 417
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "color",
        "loc": {
          "start": 418,
          "end": 423
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 418,
        "end": 423
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "label",
        "loc": {
          "start": 424,
          "end": 429
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 424,
        "end": 429
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 430,
          "end": 435
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
                "value": "Organization",
                "loc": {
                  "start": 449,
                  "end": 461
                }
              },
              "loc": {
                "start": 449,
                "end": 461
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
                    "value": "Organization_nav",
                    "loc": {
                      "start": 475,
                      "end": 491
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 472,
                    "end": 491
                  }
                }
              ],
              "loc": {
                "start": 462,
                "end": 497
              }
            },
            "loc": {
              "start": 442,
              "end": 497
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
                  "start": 509,
                  "end": 513
                }
              },
              "loc": {
                "start": 509,
                "end": 513
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
                      "start": 527,
                      "end": 535
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 524,
                    "end": 535
                  }
                }
              ],
              "loc": {
                "start": 514,
                "end": 541
              }
            },
            "loc": {
              "start": 502,
              "end": 541
            }
          }
        ],
        "loc": {
          "start": 436,
          "end": 543
        }
      },
      "loc": {
        "start": 430,
        "end": 543
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 544,
          "end": 547
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
                "start": 554,
                "end": 563
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 554,
              "end": 563
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 568,
                "end": 577
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 568,
              "end": 577
            }
          }
        ],
        "loc": {
          "start": 548,
          "end": 579
        }
      },
      "loc": {
        "start": 544,
        "end": 579
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 626,
          "end": 628
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 626,
        "end": 628
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 629,
          "end": 635
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 629,
        "end": 635
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 636,
          "end": 639
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
                "start": 646,
                "end": 659
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 646,
              "end": 659
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 664,
                "end": 673
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 664,
              "end": 673
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 678,
                "end": 689
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 678,
              "end": 689
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 694,
                "end": 703
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 694,
              "end": 703
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 708,
                "end": 717
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 708,
              "end": 717
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 722,
                "end": 729
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 722,
              "end": 729
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 734,
                "end": 746
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 734,
              "end": 746
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 751,
                "end": 759
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 751,
              "end": 759
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 764,
                "end": 778
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
                      "start": 789,
                      "end": 791
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 789,
                    "end": 791
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 800,
                      "end": 810
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 800,
                    "end": 810
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 819,
                      "end": 829
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 819,
                    "end": 829
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 838,
                      "end": 845
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 838,
                    "end": 845
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 854,
                      "end": 865
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 854,
                    "end": 865
                  }
                }
              ],
              "loc": {
                "start": 779,
                "end": 871
              }
            },
            "loc": {
              "start": 764,
              "end": 871
            }
          }
        ],
        "loc": {
          "start": 640,
          "end": 873
        }
      },
      "loc": {
        "start": 636,
        "end": 873
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 911,
          "end": 919
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
              "value": "translations",
              "loc": {
                "start": 926,
                "end": 938
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
                      "start": 949,
                      "end": 951
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 949,
                    "end": 951
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 960,
                      "end": 968
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 960,
                    "end": 968
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 977,
                      "end": 988
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 977,
                    "end": 988
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 997,
                      "end": 1001
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 997,
                    "end": 1001
                  }
                }
              ],
              "loc": {
                "start": 939,
                "end": 1007
              }
            },
            "loc": {
              "start": 926,
              "end": 1007
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1012,
                "end": 1014
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1012,
              "end": 1014
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1019,
                "end": 1029
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1019,
              "end": 1029
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1034,
                "end": 1044
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1034,
              "end": 1044
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoriesCount",
              "loc": {
                "start": 1049,
                "end": 1065
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1049,
              "end": 1065
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 1070,
                "end": 1078
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1070,
              "end": 1078
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1083,
                "end": 1092
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1083,
              "end": 1092
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 1097,
                "end": 1109
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1097,
              "end": 1109
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "runProjectsCount",
              "loc": {
                "start": 1114,
                "end": 1130
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1114,
              "end": 1130
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "simplicity",
              "loc": {
                "start": 1135,
                "end": 1145
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1135,
              "end": 1145
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 1150,
                "end": 1162
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1150,
              "end": 1162
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 1167,
                "end": 1179
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1167,
              "end": 1179
            }
          }
        ],
        "loc": {
          "start": 920,
          "end": 1181
        }
      },
      "loc": {
        "start": 911,
        "end": 1181
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1182,
          "end": 1184
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1182,
        "end": 1184
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1185,
          "end": 1195
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1185,
        "end": 1195
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 1196,
          "end": 1206
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1196,
        "end": 1206
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 1207,
          "end": 1216
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1207,
        "end": 1216
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 1217,
          "end": 1228
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1217,
        "end": 1228
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 1229,
          "end": 1235
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
                "start": 1245,
                "end": 1255
              }
            },
            "directives": [],
            "loc": {
              "start": 1242,
              "end": 1255
            }
          }
        ],
        "loc": {
          "start": 1236,
          "end": 1257
        }
      },
      "loc": {
        "start": 1229,
        "end": 1257
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 1258,
          "end": 1263
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
                "value": "Organization",
                "loc": {
                  "start": 1277,
                  "end": 1289
                }
              },
              "loc": {
                "start": 1277,
                "end": 1289
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
                    "value": "Organization_nav",
                    "loc": {
                      "start": 1303,
                      "end": 1319
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1300,
                    "end": 1319
                  }
                }
              ],
              "loc": {
                "start": 1290,
                "end": 1325
              }
            },
            "loc": {
              "start": 1270,
              "end": 1325
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
                  "start": 1337,
                  "end": 1341
                }
              },
              "loc": {
                "start": 1337,
                "end": 1341
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
                      "start": 1355,
                      "end": 1363
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1352,
                    "end": 1363
                  }
                }
              ],
              "loc": {
                "start": 1342,
                "end": 1369
              }
            },
            "loc": {
              "start": 1330,
              "end": 1369
            }
          }
        ],
        "loc": {
          "start": 1264,
          "end": 1371
        }
      },
      "loc": {
        "start": 1258,
        "end": 1371
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 1372,
          "end": 1383
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1372,
        "end": 1383
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 1384,
          "end": 1398
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1384,
        "end": 1398
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 1399,
          "end": 1404
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1399,
        "end": 1404
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 1405,
          "end": 1414
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1405,
        "end": 1414
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 1415,
          "end": 1419
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
                "start": 1429,
                "end": 1437
              }
            },
            "directives": [],
            "loc": {
              "start": 1426,
              "end": 1437
            }
          }
        ],
        "loc": {
          "start": 1420,
          "end": 1439
        }
      },
      "loc": {
        "start": 1415,
        "end": 1439
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 1440,
          "end": 1454
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1440,
        "end": 1454
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 1455,
          "end": 1460
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1455,
        "end": 1460
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1461,
          "end": 1464
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
                "start": 1471,
                "end": 1480
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1471,
              "end": 1480
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 1485,
                "end": 1496
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1485,
              "end": 1496
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 1501,
                "end": 1512
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1501,
              "end": 1512
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 1517,
                "end": 1526
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1517,
              "end": 1526
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 1531,
                "end": 1538
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1531,
              "end": 1538
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 1543,
                "end": 1551
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1543,
              "end": 1551
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 1556,
                "end": 1568
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1556,
              "end": 1568
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 1573,
                "end": 1581
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1573,
              "end": 1581
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 1586,
                "end": 1594
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1586,
              "end": 1594
            }
          }
        ],
        "loc": {
          "start": 1465,
          "end": 1596
        }
      },
      "loc": {
        "start": 1461,
        "end": 1596
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 1634,
          "end": 1642
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
              "value": "translations",
              "loc": {
                "start": 1649,
                "end": 1661
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
                      "start": 1672,
                      "end": 1674
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1672,
                    "end": 1674
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 1683,
                      "end": 1691
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1683,
                    "end": 1691
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1700,
                      "end": 1711
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1700,
                    "end": 1711
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "instructions",
                    "loc": {
                      "start": 1720,
                      "end": 1732
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1720,
                    "end": 1732
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 1741,
                      "end": 1745
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1741,
                    "end": 1745
                  }
                }
              ],
              "loc": {
                "start": 1662,
                "end": 1751
              }
            },
            "loc": {
              "start": 1649,
              "end": 1751
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1756,
                "end": 1758
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1756,
              "end": 1758
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1763,
                "end": 1773
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1763,
              "end": 1773
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1778,
                "end": 1788
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1778,
              "end": 1788
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedAt",
              "loc": {
                "start": 1793,
                "end": 1804
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1793,
              "end": 1804
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isAutomatable",
              "loc": {
                "start": 1809,
                "end": 1822
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1809,
              "end": 1822
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 1827,
                "end": 1837
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1827,
              "end": 1837
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isDeleted",
              "loc": {
                "start": 1842,
                "end": 1851
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1842,
              "end": 1851
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 1856,
                "end": 1864
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1856,
              "end": 1864
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1869,
                "end": 1878
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1869,
              "end": 1878
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "simplicity",
              "loc": {
                "start": 1883,
                "end": 1893
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1883,
              "end": 1893
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timesStarted",
              "loc": {
                "start": 1898,
                "end": 1910
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1898,
              "end": 1910
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timesCompleted",
              "loc": {
                "start": 1915,
                "end": 1929
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1915,
              "end": 1929
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "smartContractCallData",
              "loc": {
                "start": 1934,
                "end": 1955
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1934,
              "end": 1955
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "apiCallData",
              "loc": {
                "start": 1960,
                "end": 1971
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1960,
              "end": 1971
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 1976,
                "end": 1988
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1976,
              "end": 1988
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 1993,
                "end": 2005
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1993,
              "end": 2005
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 2010,
                "end": 2023
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2010,
              "end": 2023
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoryListingsCount",
              "loc": {
                "start": 2028,
                "end": 2050
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2028,
              "end": 2050
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 2055,
                "end": 2065
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2055,
              "end": 2065
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "inputsCount",
              "loc": {
                "start": 2070,
                "end": 2081
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2070,
              "end": 2081
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "nodesCount",
              "loc": {
                "start": 2086,
                "end": 2096
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2086,
              "end": 2096
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "nodeLinksCount",
              "loc": {
                "start": 2101,
                "end": 2115
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2101,
              "end": 2115
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "outputsCount",
              "loc": {
                "start": 2120,
                "end": 2132
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2120,
              "end": 2132
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 2137,
                "end": 2149
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2137,
              "end": 2149
            }
          }
        ],
        "loc": {
          "start": 1643,
          "end": 2151
        }
      },
      "loc": {
        "start": 1634,
        "end": 2151
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 2152,
          "end": 2154
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2152,
        "end": 2154
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 2155,
          "end": 2165
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2155,
        "end": 2165
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 2166,
          "end": 2176
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2166,
        "end": 2176
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isInternal",
        "loc": {
          "start": 2177,
          "end": 2187
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2177,
        "end": 2187
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 2188,
          "end": 2197
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2188,
        "end": 2197
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 2198,
          "end": 2209
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2198,
        "end": 2209
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 2210,
          "end": 2216
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
                "start": 2226,
                "end": 2236
              }
            },
            "directives": [],
            "loc": {
              "start": 2223,
              "end": 2236
            }
          }
        ],
        "loc": {
          "start": 2217,
          "end": 2238
        }
      },
      "loc": {
        "start": 2210,
        "end": 2238
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 2239,
          "end": 2244
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
                "value": "Organization",
                "loc": {
                  "start": 2258,
                  "end": 2270
                }
              },
              "loc": {
                "start": 2258,
                "end": 2270
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
                    "value": "Organization_nav",
                    "loc": {
                      "start": 2284,
                      "end": 2300
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2281,
                    "end": 2300
                  }
                }
              ],
              "loc": {
                "start": 2271,
                "end": 2306
              }
            },
            "loc": {
              "start": 2251,
              "end": 2306
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
                  "start": 2318,
                  "end": 2322
                }
              },
              "loc": {
                "start": 2318,
                "end": 2322
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
                      "start": 2336,
                      "end": 2344
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2333,
                    "end": 2344
                  }
                }
              ],
              "loc": {
                "start": 2323,
                "end": 2350
              }
            },
            "loc": {
              "start": 2311,
              "end": 2350
            }
          }
        ],
        "loc": {
          "start": 2245,
          "end": 2352
        }
      },
      "loc": {
        "start": 2239,
        "end": 2352
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 2353,
          "end": 2364
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2353,
        "end": 2364
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 2365,
          "end": 2379
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2365,
        "end": 2379
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 2380,
          "end": 2385
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2380,
        "end": 2385
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 2386,
          "end": 2395
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2386,
        "end": 2395
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 2396,
          "end": 2400
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
                "start": 2410,
                "end": 2418
              }
            },
            "directives": [],
            "loc": {
              "start": 2407,
              "end": 2418
            }
          }
        ],
        "loc": {
          "start": 2401,
          "end": 2420
        }
      },
      "loc": {
        "start": 2396,
        "end": 2420
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 2421,
          "end": 2435
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2421,
        "end": 2435
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 2436,
          "end": 2441
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2436,
        "end": 2441
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 2442,
          "end": 2445
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
                "start": 2452,
                "end": 2462
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2452,
              "end": 2462
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 2467,
                "end": 2476
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2467,
              "end": 2476
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 2481,
                "end": 2492
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2481,
              "end": 2492
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 2497,
                "end": 2506
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2497,
              "end": 2506
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 2511,
                "end": 2518
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2511,
              "end": 2518
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 2523,
                "end": 2531
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2523,
              "end": 2531
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 2536,
                "end": 2548
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2536,
              "end": 2548
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 2553,
                "end": 2561
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2553,
              "end": 2561
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 2566,
                "end": 2574
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2566,
              "end": 2574
            }
          }
        ],
        "loc": {
          "start": 2446,
          "end": 2576
        }
      },
      "loc": {
        "start": 2442,
        "end": 2576
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 2606,
          "end": 2608
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2606,
        "end": 2608
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 2609,
          "end": 2619
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2609,
        "end": 2619
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tag",
        "loc": {
          "start": 2620,
          "end": 2623
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2620,
        "end": 2623
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 2624,
          "end": 2633
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2624,
        "end": 2633
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 2634,
          "end": 2646
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
                "start": 2653,
                "end": 2655
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2653,
              "end": 2655
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 2660,
                "end": 2668
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2660,
              "end": 2668
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 2673,
                "end": 2684
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2673,
              "end": 2684
            }
          }
        ],
        "loc": {
          "start": 2647,
          "end": 2686
        }
      },
      "loc": {
        "start": 2634,
        "end": 2686
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 2687,
          "end": 2690
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
                "start": 2697,
                "end": 2702
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2697,
              "end": 2702
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 2707,
                "end": 2719
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2707,
              "end": 2719
            }
          }
        ],
        "loc": {
          "start": 2691,
          "end": 2721
        }
      },
      "loc": {
        "start": 2687,
        "end": 2721
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 2752,
          "end": 2754
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2752,
        "end": 2754
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 2755,
          "end": 2760
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2755,
        "end": 2760
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 2761,
          "end": 2765
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2761,
        "end": 2765
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 2766,
          "end": 2772
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2766,
        "end": 2772
      }
    }
  ],
  "returnType": null,
  "parentType": null,
  "schema": null,
  "fragments": {
    "Label_full": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Label_full",
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
              "value": "apisCount",
              "loc": {
                "start": 32,
                "end": 41
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 32,
              "end": 41
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "focusModesCount",
              "loc": {
                "start": 42,
                "end": 57
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 42,
              "end": 57
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 58,
                "end": 69
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 58,
              "end": 69
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "meetingsCount",
              "loc": {
                "start": 70,
                "end": 83
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 70,
              "end": 83
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "notesCount",
              "loc": {
                "start": 84,
                "end": 94
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 84,
              "end": 94
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "projectsCount",
              "loc": {
                "start": 95,
                "end": 108
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 95,
              "end": 108
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "routinesCount",
              "loc": {
                "start": 109,
                "end": 122
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 109,
              "end": 122
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "schedulesCount",
              "loc": {
                "start": 123,
                "end": 137
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 123,
              "end": 137
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "smartContractsCount",
              "loc": {
                "start": 138,
                "end": 157
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 138,
              "end": 157
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "standardsCount",
              "loc": {
                "start": 158,
                "end": 172
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 158,
              "end": 172
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 173,
                "end": 175
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 173,
              "end": 175
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 176,
                "end": 186
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 176,
              "end": 186
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 187,
                "end": 197
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 187,
              "end": 197
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "color",
              "loc": {
                "start": 198,
                "end": 203
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 198,
              "end": 203
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "label",
              "loc": {
                "start": 204,
                "end": 209
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 204,
              "end": 209
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 210,
                "end": 215
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
                      "value": "Organization",
                      "loc": {
                        "start": 229,
                        "end": 241
                      }
                    },
                    "loc": {
                      "start": 229,
                      "end": 241
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
                          "value": "Organization_nav",
                          "loc": {
                            "start": 255,
                            "end": 271
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 252,
                          "end": 271
                        }
                      }
                    ],
                    "loc": {
                      "start": 242,
                      "end": 277
                    }
                  },
                  "loc": {
                    "start": 222,
                    "end": 277
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
                        "start": 289,
                        "end": 293
                      }
                    },
                    "loc": {
                      "start": 289,
                      "end": 293
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
                            "start": 307,
                            "end": 315
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 304,
                          "end": 315
                        }
                      }
                    ],
                    "loc": {
                      "start": 294,
                      "end": 321
                    }
                  },
                  "loc": {
                    "start": 282,
                    "end": 321
                  }
                }
              ],
              "loc": {
                "start": 216,
                "end": 323
              }
            },
            "loc": {
              "start": 210,
              "end": 323
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 324,
                "end": 327
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
                      "start": 334,
                      "end": 343
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 334,
                    "end": 343
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 348,
                      "end": 357
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 348,
                    "end": 357
                  }
                }
              ],
              "loc": {
                "start": 328,
                "end": 359
              }
            },
            "loc": {
              "start": 324,
              "end": 359
            }
          }
        ],
        "loc": {
          "start": 30,
          "end": 361
        }
      },
      "loc": {
        "start": 1,
        "end": 361
      }
    },
    "Label_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Label_list",
        "loc": {
          "start": 371,
          "end": 381
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Label",
          "loc": {
            "start": 385,
            "end": 390
          }
        },
        "loc": {
          "start": 385,
          "end": 390
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
                "start": 393,
                "end": 395
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 393,
              "end": 395
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 396,
                "end": 406
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 396,
              "end": 406
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 407,
                "end": 417
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 407,
              "end": 417
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "color",
              "loc": {
                "start": 418,
                "end": 423
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 418,
              "end": 423
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "label",
              "loc": {
                "start": 424,
                "end": 429
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 424,
              "end": 429
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 430,
                "end": 435
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
                      "value": "Organization",
                      "loc": {
                        "start": 449,
                        "end": 461
                      }
                    },
                    "loc": {
                      "start": 449,
                      "end": 461
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
                          "value": "Organization_nav",
                          "loc": {
                            "start": 475,
                            "end": 491
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 472,
                          "end": 491
                        }
                      }
                    ],
                    "loc": {
                      "start": 462,
                      "end": 497
                    }
                  },
                  "loc": {
                    "start": 442,
                    "end": 497
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
                        "start": 509,
                        "end": 513
                      }
                    },
                    "loc": {
                      "start": 509,
                      "end": 513
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
                            "start": 527,
                            "end": 535
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 524,
                          "end": 535
                        }
                      }
                    ],
                    "loc": {
                      "start": 514,
                      "end": 541
                    }
                  },
                  "loc": {
                    "start": 502,
                    "end": 541
                  }
                }
              ],
              "loc": {
                "start": 436,
                "end": 543
              }
            },
            "loc": {
              "start": 430,
              "end": 543
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 544,
                "end": 547
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
                      "start": 554,
                      "end": 563
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 554,
                    "end": 563
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 568,
                      "end": 577
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 568,
                    "end": 577
                  }
                }
              ],
              "loc": {
                "start": 548,
                "end": 579
              }
            },
            "loc": {
              "start": 544,
              "end": 579
            }
          }
        ],
        "loc": {
          "start": 391,
          "end": 581
        }
      },
      "loc": {
        "start": 362,
        "end": 581
      }
    },
    "Organization_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Organization_nav",
        "loc": {
          "start": 591,
          "end": 607
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Organization",
          "loc": {
            "start": 611,
            "end": 623
          }
        },
        "loc": {
          "start": 611,
          "end": 623
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
                "start": 626,
                "end": 628
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 626,
              "end": 628
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 629,
                "end": 635
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 629,
              "end": 635
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 636,
                "end": 639
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
                      "start": 646,
                      "end": 659
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 646,
                    "end": 659
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 664,
                      "end": 673
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 664,
                    "end": 673
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 678,
                      "end": 689
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 678,
                    "end": 689
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 694,
                      "end": 703
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 694,
                    "end": 703
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 708,
                      "end": 717
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 708,
                    "end": 717
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 722,
                      "end": 729
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 722,
                    "end": 729
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 734,
                      "end": 746
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 734,
                    "end": 746
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 751,
                      "end": 759
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 751,
                    "end": 759
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 764,
                      "end": 778
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
                            "start": 789,
                            "end": 791
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 789,
                          "end": 791
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 800,
                            "end": 810
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 800,
                          "end": 810
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 819,
                            "end": 829
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 819,
                          "end": 829
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 838,
                            "end": 845
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 838,
                          "end": 845
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 854,
                            "end": 865
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 854,
                          "end": 865
                        }
                      }
                    ],
                    "loc": {
                      "start": 779,
                      "end": 871
                    }
                  },
                  "loc": {
                    "start": 764,
                    "end": 871
                  }
                }
              ],
              "loc": {
                "start": 640,
                "end": 873
              }
            },
            "loc": {
              "start": 636,
              "end": 873
            }
          }
        ],
        "loc": {
          "start": 624,
          "end": 875
        }
      },
      "loc": {
        "start": 582,
        "end": 875
      }
    },
    "Project_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Project_list",
        "loc": {
          "start": 885,
          "end": 897
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Project",
          "loc": {
            "start": 901,
            "end": 908
          }
        },
        "loc": {
          "start": 901,
          "end": 908
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
              "value": "versions",
              "loc": {
                "start": 911,
                "end": 919
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
                    "value": "translations",
                    "loc": {
                      "start": 926,
                      "end": 938
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
                            "start": 949,
                            "end": 951
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 949,
                          "end": 951
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 960,
                            "end": 968
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 960,
                          "end": 968
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 977,
                            "end": 988
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 977,
                          "end": 988
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 997,
                            "end": 1001
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 997,
                          "end": 1001
                        }
                      }
                    ],
                    "loc": {
                      "start": 939,
                      "end": 1007
                    }
                  },
                  "loc": {
                    "start": 926,
                    "end": 1007
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1012,
                      "end": 1014
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1012,
                    "end": 1014
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1019,
                      "end": 1029
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1019,
                    "end": 1029
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1034,
                      "end": 1044
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1034,
                    "end": 1044
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoriesCount",
                    "loc": {
                      "start": 1049,
                      "end": 1065
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1049,
                    "end": 1065
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 1070,
                      "end": 1078
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1070,
                    "end": 1078
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 1083,
                      "end": 1092
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1083,
                    "end": 1092
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 1097,
                      "end": 1109
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1097,
                    "end": 1109
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "runProjectsCount",
                    "loc": {
                      "start": 1114,
                      "end": 1130
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1114,
                    "end": 1130
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "simplicity",
                    "loc": {
                      "start": 1135,
                      "end": 1145
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1135,
                    "end": 1145
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 1150,
                      "end": 1162
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1150,
                    "end": 1162
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 1167,
                      "end": 1179
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1167,
                    "end": 1179
                  }
                }
              ],
              "loc": {
                "start": 920,
                "end": 1181
              }
            },
            "loc": {
              "start": 911,
              "end": 1181
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1182,
                "end": 1184
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1182,
              "end": 1184
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1185,
                "end": 1195
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1185,
              "end": 1195
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1196,
                "end": 1206
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1196,
              "end": 1206
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1207,
                "end": 1216
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1207,
              "end": 1216
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 1217,
                "end": 1228
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1217,
              "end": 1228
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 1229,
                "end": 1235
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
                      "start": 1245,
                      "end": 1255
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1242,
                    "end": 1255
                  }
                }
              ],
              "loc": {
                "start": 1236,
                "end": 1257
              }
            },
            "loc": {
              "start": 1229,
              "end": 1257
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 1258,
                "end": 1263
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
                      "value": "Organization",
                      "loc": {
                        "start": 1277,
                        "end": 1289
                      }
                    },
                    "loc": {
                      "start": 1277,
                      "end": 1289
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
                          "value": "Organization_nav",
                          "loc": {
                            "start": 1303,
                            "end": 1319
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1300,
                          "end": 1319
                        }
                      }
                    ],
                    "loc": {
                      "start": 1290,
                      "end": 1325
                    }
                  },
                  "loc": {
                    "start": 1270,
                    "end": 1325
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
                        "start": 1337,
                        "end": 1341
                      }
                    },
                    "loc": {
                      "start": 1337,
                      "end": 1341
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
                            "start": 1355,
                            "end": 1363
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1352,
                          "end": 1363
                        }
                      }
                    ],
                    "loc": {
                      "start": 1342,
                      "end": 1369
                    }
                  },
                  "loc": {
                    "start": 1330,
                    "end": 1369
                  }
                }
              ],
              "loc": {
                "start": 1264,
                "end": 1371
              }
            },
            "loc": {
              "start": 1258,
              "end": 1371
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 1372,
                "end": 1383
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1372,
              "end": 1383
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 1384,
                "end": 1398
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1384,
              "end": 1398
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 1399,
                "end": 1404
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1399,
              "end": 1404
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 1405,
                "end": 1414
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1405,
              "end": 1414
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 1415,
                "end": 1419
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
                      "start": 1429,
                      "end": 1437
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1426,
                    "end": 1437
                  }
                }
              ],
              "loc": {
                "start": 1420,
                "end": 1439
              }
            },
            "loc": {
              "start": 1415,
              "end": 1439
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 1440,
                "end": 1454
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1440,
              "end": 1454
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 1455,
                "end": 1460
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1455,
              "end": 1460
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1461,
                "end": 1464
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
                      "start": 1471,
                      "end": 1480
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1471,
                    "end": 1480
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 1485,
                      "end": 1496
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1485,
                    "end": 1496
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 1501,
                      "end": 1512
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1501,
                    "end": 1512
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1517,
                      "end": 1526
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1517,
                    "end": 1526
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1531,
                      "end": 1538
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1531,
                    "end": 1538
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 1543,
                      "end": 1551
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1543,
                    "end": 1551
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 1556,
                      "end": 1568
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1556,
                    "end": 1568
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 1573,
                      "end": 1581
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1573,
                    "end": 1581
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 1586,
                      "end": 1594
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1586,
                    "end": 1594
                  }
                }
              ],
              "loc": {
                "start": 1465,
                "end": 1596
              }
            },
            "loc": {
              "start": 1461,
              "end": 1596
            }
          }
        ],
        "loc": {
          "start": 909,
          "end": 1598
        }
      },
      "loc": {
        "start": 876,
        "end": 1598
      }
    },
    "Routine_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Routine_list",
        "loc": {
          "start": 1608,
          "end": 1620
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Routine",
          "loc": {
            "start": 1624,
            "end": 1631
          }
        },
        "loc": {
          "start": 1624,
          "end": 1631
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
              "value": "versions",
              "loc": {
                "start": 1634,
                "end": 1642
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
                    "value": "translations",
                    "loc": {
                      "start": 1649,
                      "end": 1661
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
                            "start": 1672,
                            "end": 1674
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1672,
                          "end": 1674
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 1683,
                            "end": 1691
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1683,
                          "end": 1691
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 1700,
                            "end": 1711
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1700,
                          "end": 1711
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "instructions",
                          "loc": {
                            "start": 1720,
                            "end": 1732
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1720,
                          "end": 1732
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 1741,
                            "end": 1745
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1741,
                          "end": 1745
                        }
                      }
                    ],
                    "loc": {
                      "start": 1662,
                      "end": 1751
                    }
                  },
                  "loc": {
                    "start": 1649,
                    "end": 1751
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1756,
                      "end": 1758
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1756,
                    "end": 1758
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1763,
                      "end": 1773
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1763,
                    "end": 1773
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1778,
                      "end": 1788
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1778,
                    "end": 1788
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedAt",
                    "loc": {
                      "start": 1793,
                      "end": 1804
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1793,
                    "end": 1804
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAutomatable",
                    "loc": {
                      "start": 1809,
                      "end": 1822
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1809,
                    "end": 1822
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 1827,
                      "end": 1837
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1827,
                    "end": 1837
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isDeleted",
                    "loc": {
                      "start": 1842,
                      "end": 1851
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1842,
                    "end": 1851
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 1856,
                      "end": 1864
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1856,
                    "end": 1864
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 1869,
                      "end": 1878
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1869,
                    "end": 1878
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "simplicity",
                    "loc": {
                      "start": 1883,
                      "end": 1893
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1883,
                    "end": 1893
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timesStarted",
                    "loc": {
                      "start": 1898,
                      "end": 1910
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1898,
                    "end": 1910
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timesCompleted",
                    "loc": {
                      "start": 1915,
                      "end": 1929
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1915,
                    "end": 1929
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "smartContractCallData",
                    "loc": {
                      "start": 1934,
                      "end": 1955
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1934,
                    "end": 1955
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "apiCallData",
                    "loc": {
                      "start": 1960,
                      "end": 1971
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1960,
                    "end": 1971
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 1976,
                      "end": 1988
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1976,
                    "end": 1988
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 1993,
                      "end": 2005
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1993,
                    "end": 2005
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 2010,
                      "end": 2023
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2010,
                    "end": 2023
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 2028,
                      "end": 2050
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2028,
                    "end": 2050
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 2055,
                      "end": 2065
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2055,
                    "end": 2065
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "inputsCount",
                    "loc": {
                      "start": 2070,
                      "end": 2081
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2070,
                    "end": 2081
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "nodesCount",
                    "loc": {
                      "start": 2086,
                      "end": 2096
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2086,
                    "end": 2096
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "nodeLinksCount",
                    "loc": {
                      "start": 2101,
                      "end": 2115
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2101,
                    "end": 2115
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "outputsCount",
                    "loc": {
                      "start": 2120,
                      "end": 2132
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2120,
                    "end": 2132
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 2137,
                      "end": 2149
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2137,
                    "end": 2149
                  }
                }
              ],
              "loc": {
                "start": 1643,
                "end": 2151
              }
            },
            "loc": {
              "start": 1634,
              "end": 2151
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2152,
                "end": 2154
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2152,
              "end": 2154
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 2155,
                "end": 2165
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2155,
              "end": 2165
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 2166,
                "end": 2176
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2166,
              "end": 2176
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isInternal",
              "loc": {
                "start": 2177,
                "end": 2187
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2177,
              "end": 2187
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 2188,
                "end": 2197
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2188,
              "end": 2197
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 2198,
                "end": 2209
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2198,
              "end": 2209
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 2210,
                "end": 2216
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
                      "start": 2226,
                      "end": 2236
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2223,
                    "end": 2236
                  }
                }
              ],
              "loc": {
                "start": 2217,
                "end": 2238
              }
            },
            "loc": {
              "start": 2210,
              "end": 2238
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 2239,
                "end": 2244
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
                      "value": "Organization",
                      "loc": {
                        "start": 2258,
                        "end": 2270
                      }
                    },
                    "loc": {
                      "start": 2258,
                      "end": 2270
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
                          "value": "Organization_nav",
                          "loc": {
                            "start": 2284,
                            "end": 2300
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 2281,
                          "end": 2300
                        }
                      }
                    ],
                    "loc": {
                      "start": 2271,
                      "end": 2306
                    }
                  },
                  "loc": {
                    "start": 2251,
                    "end": 2306
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
                        "start": 2318,
                        "end": 2322
                      }
                    },
                    "loc": {
                      "start": 2318,
                      "end": 2322
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
                            "start": 2336,
                            "end": 2344
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 2333,
                          "end": 2344
                        }
                      }
                    ],
                    "loc": {
                      "start": 2323,
                      "end": 2350
                    }
                  },
                  "loc": {
                    "start": 2311,
                    "end": 2350
                  }
                }
              ],
              "loc": {
                "start": 2245,
                "end": 2352
              }
            },
            "loc": {
              "start": 2239,
              "end": 2352
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 2353,
                "end": 2364
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2353,
              "end": 2364
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 2365,
                "end": 2379
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2365,
              "end": 2379
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 2380,
                "end": 2385
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2380,
              "end": 2385
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 2386,
                "end": 2395
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2386,
              "end": 2395
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 2396,
                "end": 2400
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
                      "start": 2410,
                      "end": 2418
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2407,
                    "end": 2418
                  }
                }
              ],
              "loc": {
                "start": 2401,
                "end": 2420
              }
            },
            "loc": {
              "start": 2396,
              "end": 2420
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 2421,
                "end": 2435
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2421,
              "end": 2435
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 2436,
                "end": 2441
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2436,
              "end": 2441
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2442,
                "end": 2445
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
                      "start": 2452,
                      "end": 2462
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2452,
                    "end": 2462
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 2467,
                      "end": 2476
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2467,
                    "end": 2476
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 2481,
                      "end": 2492
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2481,
                    "end": 2492
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 2497,
                      "end": 2506
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2497,
                    "end": 2506
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 2511,
                      "end": 2518
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2511,
                    "end": 2518
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 2523,
                      "end": 2531
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2523,
                    "end": 2531
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 2536,
                      "end": 2548
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2536,
                    "end": 2548
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 2553,
                      "end": 2561
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2553,
                    "end": 2561
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 2566,
                      "end": 2574
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2566,
                    "end": 2574
                  }
                }
              ],
              "loc": {
                "start": 2446,
                "end": 2576
              }
            },
            "loc": {
              "start": 2442,
              "end": 2576
            }
          }
        ],
        "loc": {
          "start": 1632,
          "end": 2578
        }
      },
      "loc": {
        "start": 1599,
        "end": 2578
      }
    },
    "Tag_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Tag_list",
        "loc": {
          "start": 2588,
          "end": 2596
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Tag",
          "loc": {
            "start": 2600,
            "end": 2603
          }
        },
        "loc": {
          "start": 2600,
          "end": 2603
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
                "start": 2606,
                "end": 2608
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2606,
              "end": 2608
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 2609,
                "end": 2619
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2609,
              "end": 2619
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tag",
              "loc": {
                "start": 2620,
                "end": 2623
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2620,
              "end": 2623
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 2624,
                "end": 2633
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2624,
              "end": 2633
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 2634,
                "end": 2646
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
                      "start": 2653,
                      "end": 2655
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2653,
                    "end": 2655
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 2660,
                      "end": 2668
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2660,
                    "end": 2668
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 2673,
                      "end": 2684
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2673,
                    "end": 2684
                  }
                }
              ],
              "loc": {
                "start": 2647,
                "end": 2686
              }
            },
            "loc": {
              "start": 2634,
              "end": 2686
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2687,
                "end": 2690
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
                      "start": 2697,
                      "end": 2702
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2697,
                    "end": 2702
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 2707,
                      "end": 2719
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2707,
                    "end": 2719
                  }
                }
              ],
              "loc": {
                "start": 2691,
                "end": 2721
              }
            },
            "loc": {
              "start": 2687,
              "end": 2721
            }
          }
        ],
        "loc": {
          "start": 2604,
          "end": 2723
        }
      },
      "loc": {
        "start": 2579,
        "end": 2723
      }
    },
    "User_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_nav",
        "loc": {
          "start": 2733,
          "end": 2741
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 2745,
            "end": 2749
          }
        },
        "loc": {
          "start": 2745,
          "end": 2749
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
                "start": 2752,
                "end": 2754
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2752,
              "end": 2754
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 2755,
                "end": 2760
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2755,
              "end": 2760
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 2761,
                "end": 2765
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2761,
              "end": 2765
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 2766,
                "end": 2772
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2766,
              "end": 2772
            }
          }
        ],
        "loc": {
          "start": 2750,
          "end": 2774
        }
      },
      "loc": {
        "start": 2724,
        "end": 2774
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
        "start": 2782,
        "end": 2799
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
              "start": 2801,
              "end": 2806
            }
          },
          "loc": {
            "start": 2800,
            "end": 2806
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
                "start": 2808,
                "end": 2835
              }
            },
            "loc": {
              "start": 2808,
              "end": 2835
            }
          },
          "loc": {
            "start": 2808,
            "end": 2836
          }
        },
        "directives": [],
        "loc": {
          "start": 2800,
          "end": 2836
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
              "start": 2842,
              "end": 2859
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 2860,
                  "end": 2865
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 2868,
                    "end": 2873
                  }
                },
                "loc": {
                  "start": 2867,
                  "end": 2873
                }
              },
              "loc": {
                "start": 2860,
                "end": 2873
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
                    "start": 2881,
                    "end": 2886
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
                          "start": 2897,
                          "end": 2903
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 2897,
                        "end": 2903
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "node",
                        "loc": {
                          "start": 2912,
                          "end": 2916
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
                                  "start": 2938,
                                  "end": 2945
                                }
                              },
                              "loc": {
                                "start": 2938,
                                "end": 2945
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
                                      "start": 2967,
                                      "end": 2979
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 2964,
                                    "end": 2979
                                  }
                                }
                              ],
                              "loc": {
                                "start": 2946,
                                "end": 2993
                              }
                            },
                            "loc": {
                              "start": 2931,
                              "end": 2993
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
                                  "start": 3013,
                                  "end": 3020
                                }
                              },
                              "loc": {
                                "start": 3013,
                                "end": 3020
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
                                      "start": 3042,
                                      "end": 3054
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 3039,
                                    "end": 3054
                                  }
                                }
                              ],
                              "loc": {
                                "start": 3021,
                                "end": 3068
                              }
                            },
                            "loc": {
                              "start": 3006,
                              "end": 3068
                            }
                          }
                        ],
                        "loc": {
                          "start": 2917,
                          "end": 3078
                        }
                      },
                      "loc": {
                        "start": 2912,
                        "end": 3078
                      }
                    }
                  ],
                  "loc": {
                    "start": 2887,
                    "end": 3084
                  }
                },
                "loc": {
                  "start": 2881,
                  "end": 3084
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "pageInfo",
                  "loc": {
                    "start": 3089,
                    "end": 3097
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
                          "start": 3108,
                          "end": 3119
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 3108,
                        "end": 3119
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorProject",
                        "loc": {
                          "start": 3128,
                          "end": 3144
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 3128,
                        "end": 3144
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorRoutine",
                        "loc": {
                          "start": 3153,
                          "end": 3169
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 3153,
                        "end": 3169
                      }
                    }
                  ],
                  "loc": {
                    "start": 3098,
                    "end": 3175
                  }
                },
                "loc": {
                  "start": 3089,
                  "end": 3175
                }
              }
            ],
            "loc": {
              "start": 2875,
              "end": 3179
            }
          },
          "loc": {
            "start": 2842,
            "end": 3179
          }
        }
      ],
      "loc": {
        "start": 2838,
        "end": 3181
      }
    },
    "loc": {
      "start": 2776,
      "end": 3181
    }
  },
  "variableValues": {},
  "path": {
    "key": "projectOrRoutine_findMany"
  }
} as const;
