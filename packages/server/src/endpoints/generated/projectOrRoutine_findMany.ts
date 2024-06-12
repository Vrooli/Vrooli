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
        "value": "versions",
        "loc": {
          "start": 240,
          "end": 248
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
                "start": 255,
                "end": 267
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
                      "start": 278,
                      "end": 280
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 278,
                    "end": 280
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 289,
                      "end": 297
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 289,
                    "end": 297
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 306,
                      "end": 317
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 306,
                    "end": 317
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 326,
                      "end": 330
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 326,
                    "end": 330
                  }
                }
              ],
              "loc": {
                "start": 268,
                "end": 336
              }
            },
            "loc": {
              "start": 255,
              "end": 336
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 341,
                "end": 343
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 341,
              "end": 343
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 348,
                "end": 358
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 348,
              "end": 358
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 363,
                "end": 373
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 363,
              "end": 373
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoriesCount",
              "loc": {
                "start": 378,
                "end": 394
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 378,
              "end": 394
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 399,
                "end": 407
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 399,
              "end": 407
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 412,
                "end": 421
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 412,
              "end": 421
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 426,
                "end": 438
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 426,
              "end": 438
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "runProjectsCount",
              "loc": {
                "start": 443,
                "end": 459
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 443,
              "end": 459
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "simplicity",
              "loc": {
                "start": 464,
                "end": 474
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 464,
              "end": 474
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 479,
                "end": 491
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 479,
              "end": 491
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 496,
                "end": 508
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 496,
              "end": 508
            }
          }
        ],
        "loc": {
          "start": 249,
          "end": 510
        }
      },
      "loc": {
        "start": 240,
        "end": 510
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 511,
          "end": 513
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 511,
        "end": 513
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 514,
          "end": 524
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 514,
        "end": 524
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 525,
          "end": 535
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 525,
        "end": 535
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 536,
          "end": 545
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 536,
        "end": 545
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 546,
          "end": 557
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 546,
        "end": 557
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 558,
          "end": 564
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
                "start": 574,
                "end": 584
              }
            },
            "directives": [],
            "loc": {
              "start": 571,
              "end": 584
            }
          }
        ],
        "loc": {
          "start": 565,
          "end": 586
        }
      },
      "loc": {
        "start": 558,
        "end": 586
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 587,
          "end": 592
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
                  "start": 606,
                  "end": 610
                }
              },
              "loc": {
                "start": 606,
                "end": 610
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
                      "start": 624,
                      "end": 632
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 621,
                    "end": 632
                  }
                }
              ],
              "loc": {
                "start": 611,
                "end": 638
              }
            },
            "loc": {
              "start": 599,
              "end": 638
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
                  "start": 650,
                  "end": 654
                }
              },
              "loc": {
                "start": 650,
                "end": 654
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
                      "start": 668,
                      "end": 676
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 665,
                    "end": 676
                  }
                }
              ],
              "loc": {
                "start": 655,
                "end": 682
              }
            },
            "loc": {
              "start": 643,
              "end": 682
            }
          }
        ],
        "loc": {
          "start": 593,
          "end": 684
        }
      },
      "loc": {
        "start": 587,
        "end": 684
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 685,
          "end": 696
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 685,
        "end": 696
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 697,
          "end": 711
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 697,
        "end": 711
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 712,
          "end": 717
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 712,
        "end": 717
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 718,
          "end": 727
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 718,
        "end": 727
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 728,
          "end": 732
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
                "start": 742,
                "end": 750
              }
            },
            "directives": [],
            "loc": {
              "start": 739,
              "end": 750
            }
          }
        ],
        "loc": {
          "start": 733,
          "end": 752
        }
      },
      "loc": {
        "start": 728,
        "end": 752
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 753,
          "end": 767
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 753,
        "end": 767
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 768,
          "end": 773
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 768,
        "end": 773
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 774,
          "end": 777
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
                "start": 784,
                "end": 793
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 784,
              "end": 793
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 798,
                "end": 809
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 798,
              "end": 809
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 814,
                "end": 825
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 814,
              "end": 825
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 830,
                "end": 839
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 830,
              "end": 839
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 844,
                "end": 851
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 844,
              "end": 851
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 856,
                "end": 864
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 856,
              "end": 864
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 869,
                "end": 881
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 869,
              "end": 881
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 886,
                "end": 894
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 886,
              "end": 894
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 899,
                "end": 907
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 899,
              "end": 907
            }
          }
        ],
        "loc": {
          "start": 778,
          "end": 909
        }
      },
      "loc": {
        "start": 774,
        "end": 909
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 947,
          "end": 955
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
                "start": 962,
                "end": 974
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
                      "start": 985,
                      "end": 987
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 985,
                    "end": 987
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 996,
                      "end": 1004
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 996,
                    "end": 1004
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1013,
                      "end": 1024
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1013,
                    "end": 1024
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "instructions",
                    "loc": {
                      "start": 1033,
                      "end": 1045
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1033,
                    "end": 1045
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 1054,
                      "end": 1058
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1054,
                    "end": 1058
                  }
                }
              ],
              "loc": {
                "start": 975,
                "end": 1064
              }
            },
            "loc": {
              "start": 962,
              "end": 1064
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1069,
                "end": 1071
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1069,
              "end": 1071
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1076,
                "end": 1086
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1076,
              "end": 1086
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1091,
                "end": 1101
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1091,
              "end": 1101
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedAt",
              "loc": {
                "start": 1106,
                "end": 1117
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1106,
              "end": 1117
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isAutomatable",
              "loc": {
                "start": 1122,
                "end": 1135
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1122,
              "end": 1135
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 1140,
                "end": 1150
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1140,
              "end": 1150
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isDeleted",
              "loc": {
                "start": 1155,
                "end": 1164
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1155,
              "end": 1164
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 1169,
                "end": 1177
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1169,
              "end": 1177
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1182,
                "end": 1191
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1182,
              "end": 1191
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "routineType",
              "loc": {
                "start": 1196,
                "end": 1207
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1196,
              "end": 1207
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "simplicity",
              "loc": {
                "start": 1212,
                "end": 1222
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1212,
              "end": 1222
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timesStarted",
              "loc": {
                "start": 1227,
                "end": 1239
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1227,
              "end": 1239
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timesCompleted",
              "loc": {
                "start": 1244,
                "end": 1258
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1244,
              "end": 1258
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 1263,
                "end": 1275
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1263,
              "end": 1275
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 1280,
                "end": 1292
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1280,
              "end": 1292
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 1297,
                "end": 1310
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1297,
              "end": 1310
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoryListingsCount",
              "loc": {
                "start": 1315,
                "end": 1337
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1315,
              "end": 1337
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 1342,
                "end": 1352
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1342,
              "end": 1352
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "inputsCount",
              "loc": {
                "start": 1357,
                "end": 1368
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1357,
              "end": 1368
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "nodesCount",
              "loc": {
                "start": 1373,
                "end": 1383
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1373,
              "end": 1383
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "nodeLinksCount",
              "loc": {
                "start": 1388,
                "end": 1402
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1388,
              "end": 1402
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "outputsCount",
              "loc": {
                "start": 1407,
                "end": 1419
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1407,
              "end": 1419
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 1424,
                "end": 1436
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1424,
              "end": 1436
            }
          }
        ],
        "loc": {
          "start": 956,
          "end": 1438
        }
      },
      "loc": {
        "start": 947,
        "end": 1438
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1439,
          "end": 1441
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1439,
        "end": 1441
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
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
        "value": "updated_at",
        "loc": {
          "start": 1453,
          "end": 1463
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1453,
        "end": 1463
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isInternal",
        "loc": {
          "start": 1464,
          "end": 1474
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1464,
        "end": 1474
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 1475,
          "end": 1484
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1475,
        "end": 1484
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
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
        "value": "labels",
        "loc": {
          "start": 1497,
          "end": 1503
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
                "start": 1513,
                "end": 1523
              }
            },
            "directives": [],
            "loc": {
              "start": 1510,
              "end": 1523
            }
          }
        ],
        "loc": {
          "start": 1504,
          "end": 1525
        }
      },
      "loc": {
        "start": 1497,
        "end": 1525
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 1526,
          "end": 1531
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
                  "start": 1545,
                  "end": 1549
                }
              },
              "loc": {
                "start": 1545,
                "end": 1549
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
                      "start": 1563,
                      "end": 1571
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1560,
                    "end": 1571
                  }
                }
              ],
              "loc": {
                "start": 1550,
                "end": 1577
              }
            },
            "loc": {
              "start": 1538,
              "end": 1577
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
                  "start": 1589,
                  "end": 1593
                }
              },
              "loc": {
                "start": 1589,
                "end": 1593
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
                      "start": 1607,
                      "end": 1615
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1604,
                    "end": 1615
                  }
                }
              ],
              "loc": {
                "start": 1594,
                "end": 1621
              }
            },
            "loc": {
              "start": 1582,
              "end": 1621
            }
          }
        ],
        "loc": {
          "start": 1532,
          "end": 1623
        }
      },
      "loc": {
        "start": 1526,
        "end": 1623
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 1624,
          "end": 1635
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1624,
        "end": 1635
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 1636,
          "end": 1650
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1636,
        "end": 1650
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 1651,
          "end": 1656
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1651,
        "end": 1656
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 1657,
          "end": 1666
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1657,
        "end": 1666
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 1667,
          "end": 1671
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
                "start": 1681,
                "end": 1689
              }
            },
            "directives": [],
            "loc": {
              "start": 1678,
              "end": 1689
            }
          }
        ],
        "loc": {
          "start": 1672,
          "end": 1691
        }
      },
      "loc": {
        "start": 1667,
        "end": 1691
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 1692,
          "end": 1706
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1692,
        "end": 1706
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 1707,
          "end": 1712
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1707,
        "end": 1712
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1713,
          "end": 1716
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
                "start": 1723,
                "end": 1733
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1723,
              "end": 1733
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 1738,
                "end": 1747
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1738,
              "end": 1747
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 1752,
                "end": 1763
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1752,
              "end": 1763
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 1768,
                "end": 1777
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1768,
              "end": 1777
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 1782,
                "end": 1789
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1782,
              "end": 1789
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 1794,
                "end": 1802
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1794,
              "end": 1802
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 1807,
                "end": 1819
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1807,
              "end": 1819
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 1824,
                "end": 1832
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1824,
              "end": 1832
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 1837,
                "end": 1845
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1837,
              "end": 1845
            }
          }
        ],
        "loc": {
          "start": 1717,
          "end": 1847
        }
      },
      "loc": {
        "start": 1713,
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
              "value": "versions",
              "loc": {
                "start": 240,
                "end": 248
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
                      "start": 255,
                      "end": 267
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
                            "start": 278,
                            "end": 280
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 278,
                          "end": 280
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 289,
                            "end": 297
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 289,
                          "end": 297
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 306,
                            "end": 317
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 306,
                          "end": 317
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 326,
                            "end": 330
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 326,
                          "end": 330
                        }
                      }
                    ],
                    "loc": {
                      "start": 268,
                      "end": 336
                    }
                  },
                  "loc": {
                    "start": 255,
                    "end": 336
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 341,
                      "end": 343
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 341,
                    "end": 343
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 348,
                      "end": 358
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 348,
                    "end": 358
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 363,
                      "end": 373
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 363,
                    "end": 373
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoriesCount",
                    "loc": {
                      "start": 378,
                      "end": 394
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 378,
                    "end": 394
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 399,
                      "end": 407
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 399,
                    "end": 407
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 412,
                      "end": 421
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 412,
                    "end": 421
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 426,
                      "end": 438
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 426,
                    "end": 438
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "runProjectsCount",
                    "loc": {
                      "start": 443,
                      "end": 459
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 443,
                    "end": 459
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "simplicity",
                    "loc": {
                      "start": 464,
                      "end": 474
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 464,
                    "end": 474
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 479,
                      "end": 491
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 479,
                    "end": 491
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 496,
                      "end": 508
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 496,
                    "end": 508
                  }
                }
              ],
              "loc": {
                "start": 249,
                "end": 510
              }
            },
            "loc": {
              "start": 240,
              "end": 510
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 511,
                "end": 513
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 511,
              "end": 513
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 514,
                "end": 524
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 514,
              "end": 524
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 525,
                "end": 535
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 525,
              "end": 535
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 536,
                "end": 545
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 536,
              "end": 545
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 546,
                "end": 557
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 546,
              "end": 557
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 558,
                "end": 564
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
                      "start": 574,
                      "end": 584
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 571,
                    "end": 584
                  }
                }
              ],
              "loc": {
                "start": 565,
                "end": 586
              }
            },
            "loc": {
              "start": 558,
              "end": 586
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 587,
                "end": 592
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
                        "start": 606,
                        "end": 610
                      }
                    },
                    "loc": {
                      "start": 606,
                      "end": 610
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
                            "start": 624,
                            "end": 632
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 621,
                          "end": 632
                        }
                      }
                    ],
                    "loc": {
                      "start": 611,
                      "end": 638
                    }
                  },
                  "loc": {
                    "start": 599,
                    "end": 638
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
                        "start": 650,
                        "end": 654
                      }
                    },
                    "loc": {
                      "start": 650,
                      "end": 654
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
                            "start": 668,
                            "end": 676
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 665,
                          "end": 676
                        }
                      }
                    ],
                    "loc": {
                      "start": 655,
                      "end": 682
                    }
                  },
                  "loc": {
                    "start": 643,
                    "end": 682
                  }
                }
              ],
              "loc": {
                "start": 593,
                "end": 684
              }
            },
            "loc": {
              "start": 587,
              "end": 684
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 685,
                "end": 696
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 685,
              "end": 696
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 697,
                "end": 711
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 697,
              "end": 711
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 712,
                "end": 717
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 712,
              "end": 717
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 718,
                "end": 727
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 718,
              "end": 727
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 728,
                "end": 732
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
                      "start": 742,
                      "end": 750
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 739,
                    "end": 750
                  }
                }
              ],
              "loc": {
                "start": 733,
                "end": 752
              }
            },
            "loc": {
              "start": 728,
              "end": 752
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 753,
                "end": 767
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 753,
              "end": 767
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 768,
                "end": 773
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 768,
              "end": 773
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 774,
                "end": 777
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
                      "start": 784,
                      "end": 793
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 784,
                    "end": 793
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 798,
                      "end": 809
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 798,
                    "end": 809
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 814,
                      "end": 825
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 814,
                    "end": 825
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 830,
                      "end": 839
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 830,
                    "end": 839
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 844,
                      "end": 851
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 844,
                    "end": 851
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 856,
                      "end": 864
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 856,
                    "end": 864
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 869,
                      "end": 881
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 869,
                    "end": 881
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 886,
                      "end": 894
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 886,
                    "end": 894
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 899,
                      "end": 907
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 899,
                    "end": 907
                  }
                }
              ],
              "loc": {
                "start": 778,
                "end": 909
              }
            },
            "loc": {
              "start": 774,
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
              "value": "versions",
              "loc": {
                "start": 947,
                "end": 955
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
                      "start": 962,
                      "end": 974
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
                            "start": 985,
                            "end": 987
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 985,
                          "end": 987
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 996,
                            "end": 1004
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 996,
                          "end": 1004
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 1013,
                            "end": 1024
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1013,
                          "end": 1024
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "instructions",
                          "loc": {
                            "start": 1033,
                            "end": 1045
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1033,
                          "end": 1045
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 1054,
                            "end": 1058
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1054,
                          "end": 1058
                        }
                      }
                    ],
                    "loc": {
                      "start": 975,
                      "end": 1064
                    }
                  },
                  "loc": {
                    "start": 962,
                    "end": 1064
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1069,
                      "end": 1071
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1069,
                    "end": 1071
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1076,
                      "end": 1086
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1076,
                    "end": 1086
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1091,
                      "end": 1101
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1091,
                    "end": 1101
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedAt",
                    "loc": {
                      "start": 1106,
                      "end": 1117
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1106,
                    "end": 1117
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAutomatable",
                    "loc": {
                      "start": 1122,
                      "end": 1135
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1122,
                    "end": 1135
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 1140,
                      "end": 1150
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1140,
                    "end": 1150
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isDeleted",
                    "loc": {
                      "start": 1155,
                      "end": 1164
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1155,
                    "end": 1164
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 1169,
                      "end": 1177
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1169,
                    "end": 1177
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 1182,
                      "end": 1191
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1182,
                    "end": 1191
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "routineType",
                    "loc": {
                      "start": 1196,
                      "end": 1207
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1196,
                    "end": 1207
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "simplicity",
                    "loc": {
                      "start": 1212,
                      "end": 1222
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1212,
                    "end": 1222
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timesStarted",
                    "loc": {
                      "start": 1227,
                      "end": 1239
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1227,
                    "end": 1239
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timesCompleted",
                    "loc": {
                      "start": 1244,
                      "end": 1258
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1244,
                    "end": 1258
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 1263,
                      "end": 1275
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1263,
                    "end": 1275
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 1280,
                      "end": 1292
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1280,
                    "end": 1292
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 1297,
                      "end": 1310
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1297,
                    "end": 1310
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 1315,
                      "end": 1337
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1315,
                    "end": 1337
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 1342,
                      "end": 1352
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1342,
                    "end": 1352
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "inputsCount",
                    "loc": {
                      "start": 1357,
                      "end": 1368
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1357,
                    "end": 1368
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "nodesCount",
                    "loc": {
                      "start": 1373,
                      "end": 1383
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1373,
                    "end": 1383
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "nodeLinksCount",
                    "loc": {
                      "start": 1388,
                      "end": 1402
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1388,
                    "end": 1402
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "outputsCount",
                    "loc": {
                      "start": 1407,
                      "end": 1419
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1407,
                    "end": 1419
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 1424,
                      "end": 1436
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1424,
                    "end": 1436
                  }
                }
              ],
              "loc": {
                "start": 956,
                "end": 1438
              }
            },
            "loc": {
              "start": 947,
              "end": 1438
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1439,
                "end": 1441
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1439,
              "end": 1441
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
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
              "value": "updated_at",
              "loc": {
                "start": 1453,
                "end": 1463
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1453,
              "end": 1463
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isInternal",
              "loc": {
                "start": 1464,
                "end": 1474
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1464,
              "end": 1474
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1475,
                "end": 1484
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1475,
              "end": 1484
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
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
              "value": "labels",
              "loc": {
                "start": 1497,
                "end": 1503
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
                      "start": 1513,
                      "end": 1523
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1510,
                    "end": 1523
                  }
                }
              ],
              "loc": {
                "start": 1504,
                "end": 1525
              }
            },
            "loc": {
              "start": 1497,
              "end": 1525
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 1526,
                "end": 1531
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
                        "start": 1545,
                        "end": 1549
                      }
                    },
                    "loc": {
                      "start": 1545,
                      "end": 1549
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
                            "start": 1563,
                            "end": 1571
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1560,
                          "end": 1571
                        }
                      }
                    ],
                    "loc": {
                      "start": 1550,
                      "end": 1577
                    }
                  },
                  "loc": {
                    "start": 1538,
                    "end": 1577
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
                        "start": 1589,
                        "end": 1593
                      }
                    },
                    "loc": {
                      "start": 1589,
                      "end": 1593
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
                            "start": 1607,
                            "end": 1615
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1604,
                          "end": 1615
                        }
                      }
                    ],
                    "loc": {
                      "start": 1594,
                      "end": 1621
                    }
                  },
                  "loc": {
                    "start": 1582,
                    "end": 1621
                  }
                }
              ],
              "loc": {
                "start": 1532,
                "end": 1623
              }
            },
            "loc": {
              "start": 1526,
              "end": 1623
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 1624,
                "end": 1635
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1624,
              "end": 1635
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 1636,
                "end": 1650
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1636,
              "end": 1650
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 1651,
                "end": 1656
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1651,
              "end": 1656
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 1657,
                "end": 1666
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1657,
              "end": 1666
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 1667,
                "end": 1671
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
                      "start": 1681,
                      "end": 1689
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1678,
                    "end": 1689
                  }
                }
              ],
              "loc": {
                "start": 1672,
                "end": 1691
              }
            },
            "loc": {
              "start": 1667,
              "end": 1691
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 1692,
                "end": 1706
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1692,
              "end": 1706
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 1707,
                "end": 1712
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1707,
              "end": 1712
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1713,
                "end": 1716
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
                      "start": 1723,
                      "end": 1733
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1723,
                    "end": 1733
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 1738,
                      "end": 1747
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1738,
                    "end": 1747
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 1752,
                      "end": 1763
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1752,
                    "end": 1763
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1768,
                      "end": 1777
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1768,
                    "end": 1777
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1782,
                      "end": 1789
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1782,
                    "end": 1789
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 1794,
                      "end": 1802
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1794,
                    "end": 1802
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 1807,
                      "end": 1819
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1807,
                    "end": 1819
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 1824,
                      "end": 1832
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1824,
                    "end": 1832
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 1837,
                      "end": 1845
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1837,
                    "end": 1845
                  }
                }
              ],
              "loc": {
                "start": 1717,
                "end": 1847
              }
            },
            "loc": {
              "start": 1713,
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
