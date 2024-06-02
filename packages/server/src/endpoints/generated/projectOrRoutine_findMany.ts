export const projectOrRoutine_findMany = {
  "fieldName": "projectOrRoutines",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "projectOrRoutines",
        "loc": {
          "start": 2501,
          "end": 2518
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 2519,
              "end": 2524
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 2527,
                "end": 2532
              }
            },
            "loc": {
              "start": 2526,
              "end": 2532
            }
          },
          "loc": {
            "start": 2519,
            "end": 2532
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
                "start": 2540,
                "end": 2545
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
                      "start": 2556,
                      "end": 2562
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2556,
                    "end": 2562
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "node",
                    "loc": {
                      "start": 2571,
                      "end": 2575
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
                              "start": 2597,
                              "end": 2604
                            }
                          },
                          "loc": {
                            "start": 2597,
                            "end": 2604
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
                                  "start": 2626,
                                  "end": 2638
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 2623,
                                "end": 2638
                              }
                            }
                          ],
                          "loc": {
                            "start": 2605,
                            "end": 2652
                          }
                        },
                        "loc": {
                          "start": 2590,
                          "end": 2652
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
                              "start": 2672,
                              "end": 2679
                            }
                          },
                          "loc": {
                            "start": 2672,
                            "end": 2679
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
                                  "start": 2701,
                                  "end": 2713
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 2698,
                                "end": 2713
                              }
                            }
                          ],
                          "loc": {
                            "start": 2680,
                            "end": 2727
                          }
                        },
                        "loc": {
                          "start": 2665,
                          "end": 2727
                        }
                      }
                    ],
                    "loc": {
                      "start": 2576,
                      "end": 2737
                    }
                  },
                  "loc": {
                    "start": 2571,
                    "end": 2737
                  }
                }
              ],
              "loc": {
                "start": 2546,
                "end": 2743
              }
            },
            "loc": {
              "start": 2540,
              "end": 2743
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "pageInfo",
              "loc": {
                "start": 2748,
                "end": 2756
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
                      "start": 2767,
                      "end": 2778
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2767,
                    "end": 2778
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorProject",
                    "loc": {
                      "start": 2787,
                      "end": 2803
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2787,
                    "end": 2803
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorRoutine",
                    "loc": {
                      "start": 2812,
                      "end": 2828
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2812,
                    "end": 2828
                  }
                }
              ],
              "loc": {
                "start": 2757,
                "end": 2834
              }
            },
            "loc": {
              "start": 2748,
              "end": 2834
            }
          }
        ],
        "loc": {
          "start": 2534,
          "end": 2838
        }
      },
      "loc": {
        "start": 2501,
        "end": 2838
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
              "value": "codeCallData",
              "loc": {
                "start": 1122,
                "end": 1134
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1122,
              "end": 1134
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isAutomatable",
              "loc": {
                "start": 1139,
                "end": 1152
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1139,
              "end": 1152
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 1157,
                "end": 1167
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1157,
              "end": 1167
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isDeleted",
              "loc": {
                "start": 1172,
                "end": 1181
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1172,
              "end": 1181
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 1186,
                "end": 1194
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1186,
              "end": 1194
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1199,
                "end": 1208
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1199,
              "end": 1208
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "simplicity",
              "loc": {
                "start": 1213,
                "end": 1223
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1213,
              "end": 1223
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timesStarted",
              "loc": {
                "start": 1228,
                "end": 1240
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1228,
              "end": 1240
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timesCompleted",
              "loc": {
                "start": 1245,
                "end": 1259
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1245,
              "end": 1259
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "apiCallData",
              "loc": {
                "start": 1264,
                "end": 1275
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1264,
              "end": 1275
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
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
              "value": "versionLabel",
              "loc": {
                "start": 1297,
                "end": 1309
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1297,
              "end": 1309
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 1314,
                "end": 1327
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1314,
              "end": 1327
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoryListingsCount",
              "loc": {
                "start": 1332,
                "end": 1354
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1332,
              "end": 1354
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 1359,
                "end": 1369
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1359,
              "end": 1369
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "inputsCount",
              "loc": {
                "start": 1374,
                "end": 1385
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1374,
              "end": 1385
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "nodesCount",
              "loc": {
                "start": 1390,
                "end": 1400
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1390,
              "end": 1400
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "nodeLinksCount",
              "loc": {
                "start": 1405,
                "end": 1419
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1405,
              "end": 1419
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "outputsCount",
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
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 1441,
                "end": 1453
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1441,
              "end": 1453
            }
          }
        ],
        "loc": {
          "start": 956,
          "end": 1455
        }
      },
      "loc": {
        "start": 947,
        "end": 1455
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1456,
          "end": 1458
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1456,
        "end": 1458
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1459,
          "end": 1469
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1459,
        "end": 1469
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 1470,
          "end": 1480
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1470,
        "end": 1480
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isInternal",
        "loc": {
          "start": 1481,
          "end": 1491
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1481,
        "end": 1491
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 1492,
          "end": 1501
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1492,
        "end": 1501
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 1502,
          "end": 1513
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1502,
        "end": 1513
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 1514,
          "end": 1520
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
                "start": 1530,
                "end": 1540
              }
            },
            "directives": [],
            "loc": {
              "start": 1527,
              "end": 1540
            }
          }
        ],
        "loc": {
          "start": 1521,
          "end": 1542
        }
      },
      "loc": {
        "start": 1514,
        "end": 1542
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 1543,
          "end": 1548
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
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Team_nav",
                    "loc": {
                      "start": 1580,
                      "end": 1588
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1577,
                    "end": 1588
                  }
                }
              ],
              "loc": {
                "start": 1567,
                "end": 1594
              }
            },
            "loc": {
              "start": 1555,
              "end": 1594
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
                  "start": 1606,
                  "end": 1610
                }
              },
              "loc": {
                "start": 1606,
                "end": 1610
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
                      "start": 1624,
                      "end": 1632
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1621,
                    "end": 1632
                  }
                }
              ],
              "loc": {
                "start": 1611,
                "end": 1638
              }
            },
            "loc": {
              "start": 1599,
              "end": 1638
            }
          }
        ],
        "loc": {
          "start": 1549,
          "end": 1640
        }
      },
      "loc": {
        "start": 1543,
        "end": 1640
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 1641,
          "end": 1652
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1641,
        "end": 1652
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 1653,
          "end": 1667
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1653,
        "end": 1667
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 1668,
          "end": 1673
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1668,
        "end": 1673
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 1674,
          "end": 1683
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1674,
        "end": 1683
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 1684,
          "end": 1688
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
                "start": 1698,
                "end": 1706
              }
            },
            "directives": [],
            "loc": {
              "start": 1695,
              "end": 1706
            }
          }
        ],
        "loc": {
          "start": 1689,
          "end": 1708
        }
      },
      "loc": {
        "start": 1684,
        "end": 1708
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 1709,
          "end": 1723
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1709,
        "end": 1723
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 1724,
          "end": 1729
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1724,
        "end": 1729
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1730,
          "end": 1733
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
                "start": 1740,
                "end": 1750
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1740,
              "end": 1750
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 1755,
                "end": 1764
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1755,
              "end": 1764
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 1769,
                "end": 1780
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1769,
              "end": 1780
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 1785,
                "end": 1794
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1785,
              "end": 1794
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 1799,
                "end": 1806
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1799,
              "end": 1806
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 1811,
                "end": 1819
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1811,
              "end": 1819
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 1824,
                "end": 1836
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1824,
              "end": 1836
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 1841,
                "end": 1849
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1841,
              "end": 1849
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 1854,
                "end": 1862
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1854,
              "end": 1862
            }
          }
        ],
        "loc": {
          "start": 1734,
          "end": 1864
        }
      },
      "loc": {
        "start": 1730,
        "end": 1864
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1894,
          "end": 1896
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1894,
        "end": 1896
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1897,
          "end": 1907
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1897,
        "end": 1907
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tag",
        "loc": {
          "start": 1908,
          "end": 1911
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1908,
        "end": 1911
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 1912,
          "end": 1921
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1912,
        "end": 1921
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 1922,
          "end": 1934
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
                "start": 1941,
                "end": 1943
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1941,
              "end": 1943
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 1948,
                "end": 1956
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1948,
              "end": 1956
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 1961,
                "end": 1972
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1961,
              "end": 1972
            }
          }
        ],
        "loc": {
          "start": 1935,
          "end": 1974
        }
      },
      "loc": {
        "start": 1922,
        "end": 1974
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1975,
          "end": 1978
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
                "start": 1985,
                "end": 1990
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1985,
              "end": 1990
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 1995,
                "end": 2007
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1995,
              "end": 2007
            }
          }
        ],
        "loc": {
          "start": 1979,
          "end": 2009
        }
      },
      "loc": {
        "start": 1975,
        "end": 2009
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 2040,
          "end": 2042
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2040,
        "end": 2042
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 2043,
          "end": 2054
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2043,
        "end": 2054
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 2055,
          "end": 2061
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2055,
        "end": 2061
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 2062,
          "end": 2074
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2062,
        "end": 2074
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 2075,
          "end": 2078
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
                "start": 2085,
                "end": 2098
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2085,
              "end": 2098
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 2103,
                "end": 2112
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2103,
              "end": 2112
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 2117,
                "end": 2128
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2117,
              "end": 2128
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 2133,
                "end": 2142
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2133,
              "end": 2142
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 2147,
                "end": 2156
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2147,
              "end": 2156
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 2161,
                "end": 2168
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2161,
              "end": 2168
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 2173,
                "end": 2185
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2173,
              "end": 2185
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 2190,
                "end": 2198
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2190,
              "end": 2198
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 2203,
                "end": 2217
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
                      "start": 2228,
                      "end": 2230
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2228,
                    "end": 2230
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 2239,
                      "end": 2249
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2239,
                    "end": 2249
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 2258,
                      "end": 2268
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2258,
                    "end": 2268
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 2277,
                      "end": 2284
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2277,
                    "end": 2284
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 2293,
                      "end": 2304
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2293,
                    "end": 2304
                  }
                }
              ],
              "loc": {
                "start": 2218,
                "end": 2310
              }
            },
            "loc": {
              "start": 2203,
              "end": 2310
            }
          }
        ],
        "loc": {
          "start": 2079,
          "end": 2312
        }
      },
      "loc": {
        "start": 2075,
        "end": 2312
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 2343,
          "end": 2345
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2343,
        "end": 2345
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 2346,
          "end": 2356
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2346,
        "end": 2356
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 2357,
          "end": 2367
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2357,
        "end": 2367
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 2368,
          "end": 2379
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2368,
        "end": 2379
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 2380,
          "end": 2386
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2380,
        "end": 2386
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 2387,
          "end": 2392
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2387,
        "end": 2392
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBotDepictingPerson",
        "loc": {
          "start": 2393,
          "end": 2413
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2393,
        "end": 2413
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 2414,
          "end": 2418
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2414,
        "end": 2418
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 2419,
          "end": 2431
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2419,
        "end": 2431
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
                    "value": "codeCallData",
                    "loc": {
                      "start": 1122,
                      "end": 1134
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1122,
                    "end": 1134
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAutomatable",
                    "loc": {
                      "start": 1139,
                      "end": 1152
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1139,
                    "end": 1152
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 1157,
                      "end": 1167
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1157,
                    "end": 1167
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isDeleted",
                    "loc": {
                      "start": 1172,
                      "end": 1181
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1172,
                    "end": 1181
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 1186,
                      "end": 1194
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1186,
                    "end": 1194
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 1199,
                      "end": 1208
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1199,
                    "end": 1208
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "simplicity",
                    "loc": {
                      "start": 1213,
                      "end": 1223
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1213,
                    "end": 1223
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timesStarted",
                    "loc": {
                      "start": 1228,
                      "end": 1240
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1228,
                    "end": 1240
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timesCompleted",
                    "loc": {
                      "start": 1245,
                      "end": 1259
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1245,
                    "end": 1259
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "apiCallData",
                    "loc": {
                      "start": 1264,
                      "end": 1275
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1264,
                    "end": 1275
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
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
                    "value": "versionLabel",
                    "loc": {
                      "start": 1297,
                      "end": 1309
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1297,
                    "end": 1309
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 1314,
                      "end": 1327
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1314,
                    "end": 1327
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 1332,
                      "end": 1354
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1332,
                    "end": 1354
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 1359,
                      "end": 1369
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1359,
                    "end": 1369
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "inputsCount",
                    "loc": {
                      "start": 1374,
                      "end": 1385
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1374,
                    "end": 1385
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "nodesCount",
                    "loc": {
                      "start": 1390,
                      "end": 1400
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1390,
                    "end": 1400
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "nodeLinksCount",
                    "loc": {
                      "start": 1405,
                      "end": 1419
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1405,
                    "end": 1419
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "outputsCount",
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
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 1441,
                      "end": 1453
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1441,
                    "end": 1453
                  }
                }
              ],
              "loc": {
                "start": 956,
                "end": 1455
              }
            },
            "loc": {
              "start": 947,
              "end": 1455
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1456,
                "end": 1458
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1456,
              "end": 1458
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1459,
                "end": 1469
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1459,
              "end": 1469
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1470,
                "end": 1480
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1470,
              "end": 1480
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isInternal",
              "loc": {
                "start": 1481,
                "end": 1491
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1481,
              "end": 1491
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1492,
                "end": 1501
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1492,
              "end": 1501
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 1502,
                "end": 1513
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1502,
              "end": 1513
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 1514,
                "end": 1520
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
                      "start": 1530,
                      "end": 1540
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1527,
                    "end": 1540
                  }
                }
              ],
              "loc": {
                "start": 1521,
                "end": 1542
              }
            },
            "loc": {
              "start": 1514,
              "end": 1542
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 1543,
                "end": 1548
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
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Team_nav",
                          "loc": {
                            "start": 1580,
                            "end": 1588
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1577,
                          "end": 1588
                        }
                      }
                    ],
                    "loc": {
                      "start": 1567,
                      "end": 1594
                    }
                  },
                  "loc": {
                    "start": 1555,
                    "end": 1594
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
                        "start": 1606,
                        "end": 1610
                      }
                    },
                    "loc": {
                      "start": 1606,
                      "end": 1610
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
                            "start": 1624,
                            "end": 1632
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1621,
                          "end": 1632
                        }
                      }
                    ],
                    "loc": {
                      "start": 1611,
                      "end": 1638
                    }
                  },
                  "loc": {
                    "start": 1599,
                    "end": 1638
                  }
                }
              ],
              "loc": {
                "start": 1549,
                "end": 1640
              }
            },
            "loc": {
              "start": 1543,
              "end": 1640
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 1641,
                "end": 1652
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1641,
              "end": 1652
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 1653,
                "end": 1667
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1653,
              "end": 1667
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 1668,
                "end": 1673
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1668,
              "end": 1673
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 1674,
                "end": 1683
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1674,
              "end": 1683
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 1684,
                "end": 1688
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
                      "start": 1698,
                      "end": 1706
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1695,
                    "end": 1706
                  }
                }
              ],
              "loc": {
                "start": 1689,
                "end": 1708
              }
            },
            "loc": {
              "start": 1684,
              "end": 1708
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 1709,
                "end": 1723
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1709,
              "end": 1723
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 1724,
                "end": 1729
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1724,
              "end": 1729
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1730,
                "end": 1733
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
                      "start": 1740,
                      "end": 1750
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1740,
                    "end": 1750
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 1755,
                      "end": 1764
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1755,
                    "end": 1764
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 1769,
                      "end": 1780
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1769,
                    "end": 1780
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1785,
                      "end": 1794
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1785,
                    "end": 1794
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1799,
                      "end": 1806
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1799,
                    "end": 1806
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 1811,
                      "end": 1819
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1811,
                    "end": 1819
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 1824,
                      "end": 1836
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1824,
                    "end": 1836
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 1841,
                      "end": 1849
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1841,
                    "end": 1849
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 1854,
                      "end": 1862
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1854,
                    "end": 1862
                  }
                }
              ],
              "loc": {
                "start": 1734,
                "end": 1864
              }
            },
            "loc": {
              "start": 1730,
              "end": 1864
            }
          }
        ],
        "loc": {
          "start": 945,
          "end": 1866
        }
      },
      "loc": {
        "start": 912,
        "end": 1866
      }
    },
    "Tag_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Tag_list",
        "loc": {
          "start": 1876,
          "end": 1884
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Tag",
          "loc": {
            "start": 1888,
            "end": 1891
          }
        },
        "loc": {
          "start": 1888,
          "end": 1891
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
                "start": 1894,
                "end": 1896
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1894,
              "end": 1896
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1897,
                "end": 1907
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1897,
              "end": 1907
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tag",
              "loc": {
                "start": 1908,
                "end": 1911
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1908,
              "end": 1911
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 1912,
                "end": 1921
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1912,
              "end": 1921
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 1922,
                "end": 1934
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
                      "start": 1941,
                      "end": 1943
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1941,
                    "end": 1943
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 1948,
                      "end": 1956
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1948,
                    "end": 1956
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1961,
                      "end": 1972
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1961,
                    "end": 1972
                  }
                }
              ],
              "loc": {
                "start": 1935,
                "end": 1974
              }
            },
            "loc": {
              "start": 1922,
              "end": 1974
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1975,
                "end": 1978
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
                      "start": 1985,
                      "end": 1990
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1985,
                    "end": 1990
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 1995,
                      "end": 2007
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1995,
                    "end": 2007
                  }
                }
              ],
              "loc": {
                "start": 1979,
                "end": 2009
              }
            },
            "loc": {
              "start": 1975,
              "end": 2009
            }
          }
        ],
        "loc": {
          "start": 1892,
          "end": 2011
        }
      },
      "loc": {
        "start": 1867,
        "end": 2011
      }
    },
    "Team_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Team_nav",
        "loc": {
          "start": 2021,
          "end": 2029
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Team",
          "loc": {
            "start": 2033,
            "end": 2037
          }
        },
        "loc": {
          "start": 2033,
          "end": 2037
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
                "start": 2040,
                "end": 2042
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2040,
              "end": 2042
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 2043,
                "end": 2054
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2043,
              "end": 2054
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 2055,
                "end": 2061
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2055,
              "end": 2061
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 2062,
                "end": 2074
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2062,
              "end": 2074
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2075,
                "end": 2078
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
                      "start": 2085,
                      "end": 2098
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2085,
                    "end": 2098
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 2103,
                      "end": 2112
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2103,
                    "end": 2112
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 2117,
                      "end": 2128
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2117,
                    "end": 2128
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 2133,
                      "end": 2142
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2133,
                    "end": 2142
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 2147,
                      "end": 2156
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2147,
                    "end": 2156
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 2161,
                      "end": 2168
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2161,
                    "end": 2168
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 2173,
                      "end": 2185
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2173,
                    "end": 2185
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 2190,
                      "end": 2198
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2190,
                    "end": 2198
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 2203,
                      "end": 2217
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
                            "start": 2228,
                            "end": 2230
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2228,
                          "end": 2230
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 2239,
                            "end": 2249
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2239,
                          "end": 2249
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 2258,
                            "end": 2268
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2258,
                          "end": 2268
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 2277,
                            "end": 2284
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2277,
                          "end": 2284
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 2293,
                            "end": 2304
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2293,
                          "end": 2304
                        }
                      }
                    ],
                    "loc": {
                      "start": 2218,
                      "end": 2310
                    }
                  },
                  "loc": {
                    "start": 2203,
                    "end": 2310
                  }
                }
              ],
              "loc": {
                "start": 2079,
                "end": 2312
              }
            },
            "loc": {
              "start": 2075,
              "end": 2312
            }
          }
        ],
        "loc": {
          "start": 2038,
          "end": 2314
        }
      },
      "loc": {
        "start": 2012,
        "end": 2314
      }
    },
    "User_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_nav",
        "loc": {
          "start": 2324,
          "end": 2332
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 2336,
            "end": 2340
          }
        },
        "loc": {
          "start": 2336,
          "end": 2340
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
                "start": 2343,
                "end": 2345
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2343,
              "end": 2345
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 2346,
                "end": 2356
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2346,
              "end": 2356
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 2357,
                "end": 2367
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2357,
              "end": 2367
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 2368,
                "end": 2379
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2368,
              "end": 2379
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 2380,
                "end": 2386
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2380,
              "end": 2386
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 2387,
                "end": 2392
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2387,
              "end": 2392
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBotDepictingPerson",
              "loc": {
                "start": 2393,
                "end": 2413
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2393,
              "end": 2413
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 2414,
                "end": 2418
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2414,
              "end": 2418
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 2419,
                "end": 2431
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2419,
              "end": 2431
            }
          }
        ],
        "loc": {
          "start": 2341,
          "end": 2433
        }
      },
      "loc": {
        "start": 2315,
        "end": 2433
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
        "start": 2441,
        "end": 2458
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
              "start": 2460,
              "end": 2465
            }
          },
          "loc": {
            "start": 2459,
            "end": 2465
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
                "start": 2467,
                "end": 2494
              }
            },
            "loc": {
              "start": 2467,
              "end": 2494
            }
          },
          "loc": {
            "start": 2467,
            "end": 2495
          }
        },
        "directives": [],
        "loc": {
          "start": 2459,
          "end": 2495
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
              "start": 2501,
              "end": 2518
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 2519,
                  "end": 2524
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 2527,
                    "end": 2532
                  }
                },
                "loc": {
                  "start": 2526,
                  "end": 2532
                }
              },
              "loc": {
                "start": 2519,
                "end": 2532
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
                    "start": 2540,
                    "end": 2545
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
                          "start": 2556,
                          "end": 2562
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 2556,
                        "end": 2562
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "node",
                        "loc": {
                          "start": 2571,
                          "end": 2575
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
                                  "start": 2597,
                                  "end": 2604
                                }
                              },
                              "loc": {
                                "start": 2597,
                                "end": 2604
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
                                      "start": 2626,
                                      "end": 2638
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 2623,
                                    "end": 2638
                                  }
                                }
                              ],
                              "loc": {
                                "start": 2605,
                                "end": 2652
                              }
                            },
                            "loc": {
                              "start": 2590,
                              "end": 2652
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
                                  "start": 2672,
                                  "end": 2679
                                }
                              },
                              "loc": {
                                "start": 2672,
                                "end": 2679
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
                                      "start": 2701,
                                      "end": 2713
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 2698,
                                    "end": 2713
                                  }
                                }
                              ],
                              "loc": {
                                "start": 2680,
                                "end": 2727
                              }
                            },
                            "loc": {
                              "start": 2665,
                              "end": 2727
                            }
                          }
                        ],
                        "loc": {
                          "start": 2576,
                          "end": 2737
                        }
                      },
                      "loc": {
                        "start": 2571,
                        "end": 2737
                      }
                    }
                  ],
                  "loc": {
                    "start": 2546,
                    "end": 2743
                  }
                },
                "loc": {
                  "start": 2540,
                  "end": 2743
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "pageInfo",
                  "loc": {
                    "start": 2748,
                    "end": 2756
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
                          "start": 2767,
                          "end": 2778
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 2767,
                        "end": 2778
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorProject",
                        "loc": {
                          "start": 2787,
                          "end": 2803
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 2787,
                        "end": 2803
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorRoutine",
                        "loc": {
                          "start": 2812,
                          "end": 2828
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 2812,
                        "end": 2828
                      }
                    }
                  ],
                  "loc": {
                    "start": 2757,
                    "end": 2834
                  }
                },
                "loc": {
                  "start": 2748,
                  "end": 2834
                }
              }
            ],
            "loc": {
              "start": 2534,
              "end": 2838
            }
          },
          "loc": {
            "start": 2501,
            "end": 2838
          }
        }
      ],
      "loc": {
        "start": 2497,
        "end": 2840
      }
    },
    "loc": {
      "start": 2435,
      "end": 2840
    }
  },
  "variableValues": {},
  "path": {
    "key": "projectOrRoutine_findMany"
  }
} as const;
